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
	mark     bool
	initial  *pdata.Metric
	previous *pdata.Metric
}

// timeseriesMap maps from a timeseries instance (metric * label values) to the timeseries info for
// the instance.
type timeseriesMapPdata struct {
	sync.RWMutex
	mark   bool
	tsiMap map[string]*timeseriesinfoPdata
}

// Get the timeseriesinfo for the timeseries associated with the metric and label values.
func (tsm *timeseriesMapPdata) get(metric *pdata.Metric, kv pdata.StringMap) *timeseriesinfoPdata {
	name := metric.Name()
	sig := getTimeseriesSignaturePdata(name, kv)
	tsi, ok := tsm.tsiMap[sig]
	if !ok {
		tsi = &timeseriesinfoPdata{}
		tsm.tsiMap[sig] = tsi
	}
	tsm.mark = true
	tsi.mark = true
	return tsi
}

// Create a unique timeseries signature consisting of the metric name and label values.
func getTimeseriesSignaturePdata(name string, kv pdata.StringMap) string {
	labelValues := make([]string, 0, kv.Len())
	kv.Sort().Range(func(_, value string) bool {
		if value != "" {
			labelValues = append(labelValues, value)
		}
		return true
	})
	return fmt.Sprintf("%s,%s", name, strings.Join(labelValues, ","))
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

// JobsMapPdata maps from a job instance to a map of timeseriesPdata instances for the job.
type JobsMapPdata struct {
	sync.RWMutex
	gcInterval time.Duration
	lastGC     time.Time
	jobsMap    map[string]*timeseriesMapPdata
}

// NewJobsMap creates a new (empty) JobsMapPdata.
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

// MetricsAdjusterPdata takes a map from a metric instance to the initial point in the metrics instance
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
func (ma *MetricsAdjusterPdata) AdjustMetrics(metricL *pdata.MetricSlice) int {
	resets := 0
	ma.tsm.Lock()
	defer ma.tsm.Unlock()
	for i := 0; i < metricL.Len(); i++ {
		metric := metricL.At(i)
		resets += ma.adjustMetric(&metric)
	}
	return resets
}

// Returns the number of timeseries with reset start times.
func (ma *MetricsAdjusterPdata) adjustMetric(metric *pdata.Metric) int {
	switch metric.DataType() {
	case pdata.MetricDataTypeGauge:
		// gauges don't need to be adjusted so no additional processing is necessary
		return 0
	default:
		return ma.adjustMetricPoints(metric)
	}
}

// Returns  the number of timeseries that had reset start times.
func (ma *MetricsAdjusterPdata) adjustMetricPoints(metric *pdata.Metric) int {
	switch dataType := metric.DataType(); dataType {
	case pdata.MetricDataTypeGauge:
		return ma.adjustMetricGauge(metric)

	case pdata.MetricDataTypeHistogram:
		// No need to adjust Gauge-Histograms.
		// TODO(@odeke-em): examine and see if perhaps the AggregationTemporality
		// changes whether we need to adjust the metric. Currently the prior tests
		return 0

	case pdata.MetricDataTypeSummary:
		return ma.adjustMetricSummary(metric)

	case pdata.MetricDataTypeSum:
		return ma.adjustMetricSum(metric)

	default:
		// this shouldn't happen
		ma.logger.Info("Adjust - skipping unexpected point", zap.String("type", dataType.String()))
		return 0
	}
}

// Returns true if 'current' was adjusted and false if 'current' is an the initial occurrence or a
// reset of the timeseries.
func (ma *MetricsAdjusterPdata) adjustMetricGauge(current *pdata.Metric) (resets int) {
	currentPoints := current.Gauge().DataPoints()

	for i := 0; i < currentPoints.Len(); i++ {
		currentGauge := currentPoints.At(i)
		tsi := ma.tsm.get(current, currentGauge.LabelsMap())
		previous := tsi.previous
		tsi.previous = current
		if tsi.initial == nil {
			// initial || reset timeseries.
			tsi.initial = current
			resets++
		}
		initialPoints := tsi.initial.Gauge().DataPoints()
		previousPoints := previous.Gauge().DataPoints()
		if i >= initialPoints.Len() || i >= previousPoints.Len() {
			ma.logger.Info("Adjusting Points, all lengths should be equal",
				zap.Int("len(current)", currentPoints.Len()),
				zap.Int("len(initial)", initialPoints.Len()),
				zap.Int("len(previous)", previousPoints.Len()))
			// initial || reset timeseries.
			tsi.initial = current
			resets++
			continue
		}

		currentGauge, previousGauge := currentPoints.At(i), previousPoints.At(i)
		if currentGauge.DoubleVal() < previousGauge.DoubleVal() {
			// reset detected
			tsi.initial = current
			continue
		}
		initialGauge := initialPoints.At(i)
		currentGauge.SetStartTimestamp(initialGauge.StartTimestamp())
		resets++
	}
	return
}

// Adding this here to preserve the code, given that golangc-lint is so pedantic.
// We'll need the code in this method when we deal with cumulative vs gauge distributions.
var _ = (*MetricsAdjusterPdata)(nil).adjustMetricHistogram

func (ma *MetricsAdjusterPdata) adjustMetricHistogram(current *pdata.Metric) (resets int) {
	// note: sum of squared deviation not currently supported
	currentPoints := current.Histogram().DataPoints()

	for i := 0; i < currentPoints.Len(); i++ {
		currentDist := currentPoints.At(i)
		tsi := ma.tsm.get(current, currentDist.LabelsMap())
		previous := tsi.previous
		tsi.previous = current
		if tsi.initial == nil {
			// initial || reset timeseries.
			tsi.initial = current
			resets++
			continue
		}
		initialPoints := tsi.initial.Histogram().DataPoints()
		previousPoints := previous.Histogram().DataPoints()
		if i >= initialPoints.Len() || i >= previousPoints.Len() {
			ma.logger.Info("Adjusting Points, all lengths should be equal",
				zap.Int("len(current)", currentPoints.Len()),
				zap.Int("len(initial)", initialPoints.Len()),
				zap.Int("len(previous)", previousPoints.Len()))
			// initial || reset timeseries.
			tsi.initial = current
			resets++
			continue
		}

		previousDist := previousPoints.At(i)
		if currentDist.Count() < previousDist.Count() || currentDist.Sum() < previousDist.Sum() {
			// reset detected
			tsi.initial = current
			resets++
			continue
		}
		initialDist := initialPoints.At(i)
		currentDist.SetStartTimestamp(initialDist.StartTimestamp())
	}
	return
}

func (ma *MetricsAdjusterPdata) adjustMetricSum(current *pdata.Metric) (resets int) {
	currentPoints := current.Sum().DataPoints()

	for i := 0; i < currentPoints.Len(); i++ {
		currentSum := currentPoints.At(i)
		tsi := ma.tsm.get(current, currentSum.LabelsMap())
		previous := tsi.previous
		tsi.previous = current
		if tsi.initial == nil {
			// initial || reset timeseries.
			tsi.initial = current
			resets++
			continue
		}
		initialPoints := tsi.initial.Sum().DataPoints()
		previousPoints := previous.Sum().DataPoints()
		if i >= initialPoints.Len() || i >= previousPoints.Len() {
			ma.logger.Info("Adjusting Points, all lengths should be equal",
				zap.Int("len(current)", currentPoints.Len()),
				zap.Int("len(initial)", initialPoints.Len()),
				zap.Int("len(previous)", previousPoints.Len()))
			tsi.initial = current
			resets++
			continue
		}

		previousSum := previousPoints.At(i)
		if currentSum.DoubleVal() < previousSum.DoubleVal() {
			// reset detected
			tsi.initial = current
			resets++
			continue
		}
		initialSum := initialPoints.At(i)
		currentSum.SetStartTimestamp(initialSum.StartTimestamp())
	}

	return
}

func (ma *MetricsAdjusterPdata) adjustMetricSummary(current *pdata.Metric) (resets int) {
	currentPoints := current.Summary().DataPoints()

	for i := 0; i < currentPoints.Len(); i++ {
		currentSummary := currentPoints.At(i)
		tsi := ma.tsm.get(current, currentSummary.LabelsMap())
		previous := tsi.previous
		tsi.previous = current
		if tsi.initial == nil {
			// initial || reset timeseries.
			tsi.initial = current
			resets++
			continue
		}
		initialPoints := tsi.initial.Summary().DataPoints()
		previousPoints := previous.Summary().DataPoints()
		if i >= initialPoints.Len() || i >= previousPoints.Len() {
			ma.logger.Info("Adjusting Points, all lengths should be equal",
				zap.Int("len(current)", currentPoints.Len()),
				zap.Int("len(initial)", initialPoints.Len()),
				zap.Int("len(previous)", previousPoints.Len()))
			tsi.initial = current
			resets++
			continue
		}

		previousSummary := previousPoints.At(i)
		if (currentSummary.Count() != 0 &&
			previousSummary.Count() != 0 &&
			currentSummary.Count() < previousSummary.Count()) ||

			(currentSummary.Sum() != 0 &&
				previousSummary.Sum() != 0 &&
				currentSummary.Sum() < previousSummary.Sum()) {
			// reset detected
			tsi.initial = current
			resets++
			continue
		}
		initialSummary := initialPoints.At(i)
		currentSummary.SetStartTimestamp(initialSummary.StartTimestamp())
	}

	return
}
