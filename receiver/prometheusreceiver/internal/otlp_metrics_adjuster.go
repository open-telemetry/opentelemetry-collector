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
	"go.opentelemetry.io/collector/model/pdata"
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
type timeseriesinfoPdata struct {
	timeseriesinfo
	mark     bool
	initial  *metricspb.TimeSeries
	previous *metricspb.TimeSeries
}

// timeseriesMap maps from a timeseries instance (metric * label values) to the timeseries info for
// the instance.
type timeseriesMapPdata struct {
	sync.RWMutex
	mark   bool
	tsiMap map[string]*timeseriesinfoPdata
}

// Get the timeseriesinfo for the timeseries associated with the metric and label values.
func (tsm *timeseriesMapPdata) get(metric *metricspb.Metric, values []*metricspb.LabelValue) *timeseriesinfoPdata {
	name := metric.GetMetricDescriptor().GetName()
	sig := getTimeseriesSignature(name, values)
	tsi, ok := tsm.tsiMap[sig]
	if !ok {
		tsi = &timeseriesinfoPdata{}
		tsm.tsiMap[sig] = tsi
	}
	tsm.mark = true
	tsi.mark = true
	return tsi
}

// Remove timeseries that have aged out.
func (tsm *timeseriesMapPdata) gc() {
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

func newTimeseriesMapPdata() *timeseriesMapPdata {
	return &timeseriesMapPdata{mark: true, tsiMap: map[string]*timeseriesinfoPdata{}}
}

// Create a unique timeseries signature consisting of the metric name and label values.
func getTimeseriesSignaturePdata(name string, values []*metricspb.LabelValue) string {
	labelValues := make([]string, 0, len(values))
	for _, label := range values {
		if label.GetValue() != "" {
			labelValues = append(labelValues, label.GetValue())
		}
	}
	return fmt.Sprintf("%s,%s", name, strings.Join(labelValues, ","))
}

// JobsMapPdata maps from a job instance to a map of timeseries instances for the job.
type JobsMapPdata struct {
	sync.RWMutex
	gcInterval time.Duration
	lastGC     time.Time
	jobsMap    map[string]*timeseriesMapPdata
}

// NewJobsMap creates a new (empty) JobsMap.
func NewJobsMapPdata(gcInterval time.Duration) *JobsMapPdata {
	return &JobsMapPdata{gcInterval: gcInterval, lastGC: time.Now(), jobsMap: make(map[string]*timeseriesMapPdata)}
}

// Remove jobs and timeseries that have aged out.
func (jm *JobsMapPdata) gc() {
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

func (jm *JobsMapPdata) maybeGC() {
	// speculatively check if gc() is necessary, recheck once the structure is locked
	jm.RLock()
	defer jm.RUnlock()
	if time.Since(jm.lastGC) > jm.gcInterval {
		go jm.gc()
	}
}

func (jm *JobsMapPdata) get(job, instance string) *timeseriesMapPdata {
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
	tsm2 = newTimeseriesMapPdata()
	jm.jobsMap[sig] = tsm2
	return tsm2
}

// MetricsAdjuster takes a map from a metric instance to the initial point in the metrics instance
// and provides AdjustMetrics, which takes a sequence of metrics and adjust their start times based on
// the initial points.
type MetricsAdjusterPdata struct {
	tsm    *timeseriesMapPdata
	logger *zap.Logger
}

// NewMetricsAdjuster is a constructor for MetricsAdjuster.
func NewMetricsAdjusterPdata(tsm *timeseriesMapPdata, logger *zap.Logger) *MetricsAdjusterPdata {
	return &MetricsAdjusterPdata{
		tsm:    tsm,
		logger: logger,
	}
}

// AdjustMetrics takes a sequence of metrics and adjust their start times based on the initial and
// previous points in the timeseriesMap.
// Returns the total number of timeseries that had reset start times.
func (ma *MetricsAdjusterPdata) AdjustMetricsPdata(metrics []*pdata.Metric) ([]*pdata.Metric, int) {
	var adjusted = make([]*pdata.Metric, 0, len(metrics))
	resets := 0
	ma.tsm.Lock()
	defer ma.tsm.Unlock()
	for _, metric := range metrics {
		d := ma.adjustMetricPdata(metric)
		resets += d
		adjusted = append(adjusted, metric)
	}
	return adjusted, resets
}

// Returns the number of timeseries with reset start times.
//
// Types of metrics returned supported by prometheus:
// - pdata.MetricDataTypeDoubleGauge
// - pdata.MetricDataTypeIntHistogram
// - pdata.MetricDataTypeDoubleSum
// - pdata.MetricDataTypeHistogram
// - pdata.MetricDataTypeSummary
func (ma *MetricsAdjusterPdata) adjustMetricPdata(metric *pdata.Metric) int {
	switch metric.DataType() {
	case pdata.MetricDataTypeDoubleGauge, pdata.MetricDataTypeIntHistogram:
		// gauges don't need to be adjusted so no additional processing is necessary
		return 0
	default:
		return ma.adjustMetricTimeseries(metric)
	}
}

func pdataMetricLen(metric *pdata.Metric) int {
	switch dataType := metric.DataType(); dataType {
	case pdata.MetricDataTypeIntGauge:
		return metric.IntGauge().DataPoints().Len()
	case pdata.MetricDataTypeDoubleGauge:
		return metric.DoubleGauge().DataPoints().Len()
	case pdata.MetricDataTypeIntSum:
		return metric.IntSum().DataPoints().Len()
	case pdata.MetricDataTypeDoubleSum:
		return metric.DoubleSum().DataPoints().Len()
	case pdata.MetricDataTypeIntHistogram:
		return metric.IntHistogram().DataPoints().Len()
	case pdata.MetricDataTypeHistogram:
		return metric.Histogram().DataPoints().Len()
	case pdata.MetricDataTypeSummary:
		return metric.Summary().DataPoints().Len()
	default:
		// Intentionally returning -1 so that we crash loudly instead
		// of silently passing.
		panic(fmt.Sprintf("Cannot determine the length of a unknown type: %d::%s", dataType, dataType))
	}
}

// Returns  the number of timeseries that had reset start times.
func (ma *MetricsAdjusterPdata) adjustMetricTimeseries(metric *pdata.Metric) int {
	// TODO (@odeke-em): Fill me in.
	return -1
}

// Returns true if 'current' was adjusted and false if 'current' is an the initial occurrence or a
// reset of the timeseries.
func (ma *MetricsAdjusterPdata) adjustTimeseries(mType pdata.MetricDataType, current, initial, previous *metricspb.TimeSeries) bool {
	// TODO (@odeke-em): Fill me in.
	return false
}

func (ma *MetricsAdjusterPdata) adjustPoints(metricType pdata.MetricDataType, current, initial, previous []*metricspb.Point) bool {
	if len(current) != 1 || len(initial) != 1 || len(current) != 1 {
		ma.logger.Info("Adjusting Points, all lengths should be 1",
			zap.Int("len(current)", len(current)), zap.Int("len(initial)", len(initial)), zap.Int("len(previous)", len(previous)))
		return true
	}
	return ma.isReset(metricType, current[0], previous[0])
}

func (ma *MetricsAdjusterPdata) isReset(metricType pdata.MetricDataType, current, previous *metricspb.Point) bool {
	// TODO (@odeke-em): Fill me in.
	return false
}
