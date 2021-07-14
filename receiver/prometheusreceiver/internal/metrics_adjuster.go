// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"fmt"
	"strings"
	"sync"
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"go.uber.org/zap"
)

// Notes on garbage collection (gc):
//
// Job-level gc:
// The Prometheus receiver will likely execute in a long running service whose lifetime may exceed
// the lifetimes of many of the jobs that it is collecting from. In order to keep the JobsMap from
// leaking memory for entries of no-longer existing jobs, the JobsMap needs to remove entries that
// haven't been accessed for a long period of time.
//
// Timeseries-level gc:
// Some jobs that the Prometheus receiver is collecting from may export timeseries based on metrics
// from other jobs (e.g. cAdvisor). In order to keep the timeseriesMap from leaking memory for entries
// of no-longer existing jobs, the timeseriesMap for each job needs to remove entries that haven't
// been accessed for a long period of time.
//
// The gc strategy uses a standard mark-and-sweep approach - each time a timeseriesMap is accessed,
// it is marked. Similarly, each time a timeseriesinfo is accessed, it is also marked.
//
// At the end of each JobsMap.get(), if the last time the JobsMap was gc'd exceeds the 'gcInterval',
// the JobsMap is locked and any timeseriesMaps that are unmarked are removed from the JobsMap
// otherwise the timeseriesMap is gc'd
//
// The gc for the timeseriesMap is straightforward - the map is locked and, for each timeseriesinfo
// in the map, if it has not been marked, it is removed otherwise it is unmarked.
//
// Alternative Strategies
// 1. If the job-level gc doesn't run often enough, or runs too often, a separate go routine can
//    be spawned at JobMap creation time that gc's at periodic intervals. This approach potentially
//    adds more contention and latency to each scrape so the current approach is used. Note that
//    the go routine will need to be cancelled upon Shutdown().
// 2. If the gc of each timeseriesMap during the gc of the JobsMap causes too much contention,
//    the gc of timeseriesMaps can be moved to the end of MetricsAdjuster().AdjustMetrics(). This
//    approach requires adding 'lastGC' Time and (potentially) a gcInterval duration to
//    timeseriesMap so the current approach is used instead.

// timeseriesinfo contains the information necessary to adjust from the initial point and to detect
// resets.
type timeseriesinfo struct {
	mark     bool
	initial  *metricspb.TimeSeries
	previous *metricspb.TimeSeries
}

// timeseriesMap maps from a timeseries instance (metric * label values) to the timeseries info for
// the instance.
type timeseriesMap struct {
	sync.RWMutex
	mark   bool
	tsiMap map[string]*timeseriesinfo
}

// Get the timeseriesinfo for the timeseries associated with the metric and label values.
func (tsm *timeseriesMap) get(
	metric *metricspb.Metric, values []*metricspb.LabelValue) *timeseriesinfo {
	name := metric.GetMetricDescriptor().GetName()
	sig := getTimeseriesSignature(name, values)
	tsi, ok := tsm.tsiMap[sig]
	if !ok {
		tsi = &timeseriesinfo{}
		tsm.tsiMap[sig] = tsi
	}
	tsm.mark = true
	tsi.mark = true
	return tsi
}

// Remove timeseries that have aged out.
func (tsm *timeseriesMap) gc() {
	tsm.Lock()
	defer tsm.Unlock()
	// this shouldn't happen under the current gc() strategy
	if !tsm.mark {
		return
	}
	for ts, tsi := range tsm.tsiMap {
		if !tsi.mark {
			delete(tsm.tsiMap, ts)
		} else {
			tsi.mark = false
		}
	}
	tsm.mark = false
}

func newTimeseriesMap() *timeseriesMap {
	return &timeseriesMap{mark: true, tsiMap: map[string]*timeseriesinfo{}}
}

// Create a unique timeseries signature consisting of the metric name and label values.
func getTimeseriesSignature(name string, values []*metricspb.LabelValue) string {
	labelValues := make([]string, 0, len(values))
	for _, label := range values {
		if label.GetValue() != "" {
			labelValues = append(labelValues, label.GetValue())
		}
	}
	return fmt.Sprintf("%s,%s", name, strings.Join(labelValues, ","))
}

// JobsMap maps from a job instance to a map of timeseries instances for the job.
type JobsMap struct {
	sync.RWMutex
	gcInterval time.Duration
	lastGC     time.Time
	jobsMap    map[string]*timeseriesMap
}

// NewJobsMap creates a new (empty) JobsMap.
func NewJobsMap(gcInterval time.Duration) *JobsMap {
	return &JobsMap{gcInterval: gcInterval, lastGC: time.Now(), jobsMap: make(map[string]*timeseriesMap)}
}

// Remove jobs and timeseries that have aged out.
func (jm *JobsMap) gc() {
	jm.Lock()
	defer jm.Unlock()
	// once the structure is locked, confirm that gc() is still necessary
	if time.Since(jm.lastGC) > jm.gcInterval {
		for sig, tsm := range jm.jobsMap {
			tsm.RLock()
			tsmNotMarked := !tsm.mark
			tsm.RUnlock()
			if tsmNotMarked {
				delete(jm.jobsMap, sig)
			} else {
				tsm.gc()
			}
		}
		jm.lastGC = time.Now()
	}
}

func (jm *JobsMap) maybeGC() {
	// speculatively check if gc() is necessary, recheck once the structure is locked
	jm.RLock()
	defer jm.RUnlock()
	if time.Since(jm.lastGC) > jm.gcInterval {
		go jm.gc()
	}
}

func (jm *JobsMap) get(job, instance string) *timeseriesMap {
	sig := job + ":" + instance
	jm.RLock()
	tsm, ok := jm.jobsMap[sig]
	jm.RUnlock()
	defer jm.maybeGC()
	if ok {
		return tsm
	}
	jm.Lock()
	defer jm.Unlock()
	tsm2, ok2 := jm.jobsMap[sig]
	if ok2 {
		return tsm2
	}
	tsm2 = newTimeseriesMap()
	jm.jobsMap[sig] = tsm2
	return tsm2
}

// MetricsAdjuster takes a map from a metric instance to the initial point in the metrics instance
// and provides AdjustMetrics, which takes a sequence of metrics and adjust their start times based on
// the initial points.
type MetricsAdjuster struct {
	tsm    *timeseriesMap
	logger *zap.Logger
}

// NewMetricsAdjuster is a constructor for MetricsAdjuster.
func NewMetricsAdjuster(tsm *timeseriesMap, logger *zap.Logger) *MetricsAdjuster {
	return &MetricsAdjuster{
		tsm:    tsm,
		logger: logger,
	}
}

// AdjustMetrics takes a sequence of metrics and adjust their start times based on the initial and
// previous points in the timeseriesMap.
// Returns the total number of timeseries that had reset start times.
func (ma *MetricsAdjuster) AdjustMetrics(metrics []*metricspb.Metric) ([]*metricspb.Metric, int) {
	var adjusted = make([]*metricspb.Metric, 0, len(metrics))
	resets := 0
	ma.tsm.Lock()
	defer ma.tsm.Unlock()
	for _, metric := range metrics {
		d := ma.adjustMetric(metric)
		resets += d
		adjusted = append(adjusted, metric)
	}
	return adjusted, resets
}

// Returns the number of timeseries with reset start times.
//
// Types of metrics returned supported by prometheus:
// - MetricDescriptor_GAUGE_DOUBLE
// - MetricDescriptor_GAUGE_DISTRIBUTION
// - MetricDescriptor_CUMULATIVE_DOUBLE
// - MetricDescriptor_CUMULATIVE_DISTRIBUTION
// - MetricDescriptor_SUMMARY
func (ma *MetricsAdjuster) adjustMetric(metric *metricspb.Metric) int {
	switch metric.MetricDescriptor.Type {
	case metricspb.MetricDescriptor_GAUGE_DOUBLE, metricspb.MetricDescriptor_GAUGE_DISTRIBUTION:
		// gauges don't need to be adjusted so no additional processing is necessary
		return 0
	default:
		return ma.adjustMetricTimeseries(metric)
	}
}

// Returns  the number of timeseries that had reset start times.
func (ma *MetricsAdjuster) adjustMetricTimeseries(metric *metricspb.Metric) int {
	resets := 0
	filtered := make([]*metricspb.TimeSeries, 0, len(metric.GetTimeseries()))
	for _, current := range metric.GetTimeseries() {
		tsi := ma.tsm.get(metric, current.GetLabelValues())
		if tsi.initial == nil || !ma.adjustTimeseries(metric.MetricDescriptor.Type, current, tsi.initial, tsi.previous) {
			// initial || reset timeseries
			tsi.initial = current
			resets++
		}
		tsi.previous = current
		filtered = append(filtered, current)
	}
	metric.Timeseries = filtered
	return resets
}

// Returns true if 'current' was adjusted and false if 'current' is an the initial occurrence or a
// reset of the timeseries.
func (ma *MetricsAdjuster) adjustTimeseries(metricType metricspb.MetricDescriptor_Type,
	current, initial, previous *metricspb.TimeSeries) bool {
	if !ma.adjustPoints(
		metricType, current.GetPoints(), initial.GetPoints(), previous.GetPoints()) {
		return false
	}
	current.StartTimestamp = initial.StartTimestamp
	return true
}

func (ma *MetricsAdjuster) adjustPoints(metricType metricspb.MetricDescriptor_Type,
	current, initial, previous []*metricspb.Point) bool {
	if len(current) != 1 || len(initial) != 1 || len(previous) != 1 {
		ma.logger.Info("Adjusting Points, all lengths should be 1",
			zap.Int("len(current)", len(current)), zap.Int("len(initial)", len(initial)), zap.Int("len(previous)", len(previous)))
		return true
	}
	return ma.isReset(metricType, current[0], previous[0])
}

func (ma *MetricsAdjuster) isReset(metricType metricspb.MetricDescriptor_Type,
	current, previous *metricspb.Point) bool {
	switch metricType {
	case metricspb.MetricDescriptor_CUMULATIVE_DOUBLE:
		if current.GetDoubleValue() < previous.GetDoubleValue() {
			// reset detected
			return false
		}
	case metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION:
		// note: sum of squared deviation not currently supported
		currentDist := current.GetDistributionValue()
		previousDist := previous.GetDistributionValue()
		if currentDist == nil || previousDist == nil {
			return false
		}
		if currentDist.Count < previousDist.Count || currentDist.Sum < previousDist.Sum {
			// reset detected
			return false
		}
	case metricspb.MetricDescriptor_SUMMARY:
		currentSummary := current.GetSummaryValue()
		previousSummary := previous.GetSummaryValue()
		if currentSummary == nil || previousSummary == nil {
			return false
		}
		if (currentSummary.Count != nil &&
			previousSummary.Count != nil &&
			currentSummary.Count.GetValue() < previousSummary.Count.GetValue()) ||

			(currentSummary.Sum != nil &&
				previousSummary.Sum != nil &&
				currentSummary.Sum.GetValue() < previousSummary.Sum.GetValue()) {
			// reset detected
			return false
		}
	default:
		// this shouldn't happen
		ma.logger.Info("Adjust - skipping unexpected point", zap.String("type", metricType.String()))
	}
	return true
}
