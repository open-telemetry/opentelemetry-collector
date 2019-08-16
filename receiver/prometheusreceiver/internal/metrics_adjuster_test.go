// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"reflect"
	"testing"
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/golang/protobuf/ptypes/wrappers"
	"go.uber.org/zap"
)

func Test_gauge(t *testing.T) {
	script := []*metricsAdjusterTest{{
		"Gauge: round 1 - gauge not adjusted",
		[]*metricspb.Metric{gauge(k1k2, timeseries(1, v1v2, double(1, 44)))},
		[]*metricspb.Metric{gauge(k1k2, timeseries(1, v1v2, double(1, 44)))},
	}, {
		"Gauge: round 2 - gauge not adjusted",
		[]*metricspb.Metric{gauge(k1k2, timeseries(2, v1v2, double(2, 66)))},
		[]*metricspb.Metric{gauge(k1k2, timeseries(2, v1v2, double(2, 66)))},
	}, {
		"Gauge: round 3 - value less than previous value - gauge is not adjusted",
		[]*metricspb.Metric{gauge(k1k2, timeseries(3, v1v2, double(3, 55)))},
		[]*metricspb.Metric{gauge(k1k2, timeseries(3, v1v2, double(3, 55)))},
	}}
	runScript(t, NewJobsMap(time.Duration(time.Minute)).get("job", "0"), script)
}

func Test_gaugeDistribution(t *testing.T) {
	script := []*metricsAdjusterTest{{
		"GaugeDist: round 1 - gauge distribution not adjusted",
		[]*metricspb.Metric{gaugeDist(k1k2, timeseries(1, v1v2, dist(1, bounds0, []int64{4, 2, 3, 7})))},
		[]*metricspb.Metric{gaugeDist(k1k2, timeseries(1, v1v2, dist(1, bounds0, []int64{4, 2, 3, 7})))},
	}, {
		"GaugeDist: round 2 - gauge distribution not adjusted",
		[]*metricspb.Metric{gaugeDist(k1k2, timeseries(2, v1v2, dist(2, bounds0, []int64{6, 5, 8, 11})))},
		[]*metricspb.Metric{gaugeDist(k1k2, timeseries(2, v1v2, dist(2, bounds0, []int64{6, 5, 8, 11})))},
	}, {
		"GaugeDist: round 3 - count/sum less than previous - gauge distribution not adjusted",
		[]*metricspb.Metric{gaugeDist(k1k2, timeseries(3, v1v2, dist(3, bounds0, []int64{2, 0, 1, 5})))},
		[]*metricspb.Metric{gaugeDist(k1k2, timeseries(3, v1v2, dist(3, bounds0, []int64{2, 0, 1, 5})))},
	}}
	runScript(t, NewJobsMap(time.Duration(time.Minute)).get("job", "0"), script)
}

func Test_cumulative(t *testing.T) {
	script := []*metricsAdjusterTest{{
		"Cumulative: round 1 - initial instance, adjusted should be empty",
		[]*metricspb.Metric{cumulative(k1k2, timeseries(1, v1v2, double(1, 44)))},
		[]*metricspb.Metric{},
	}, {
		"Cumulative: round 2 - instance adjusted based on round 1",
		[]*metricspb.Metric{cumulative(k1k2, timeseries(2, v1v2, double(2, 66)))},
		[]*metricspb.Metric{cumulative(k1k2, timeseries(1, v1v2, double(2, 22)))},
	}, {
		"Cumulative: round 3 - instance reset (value less than previous value), adjusted should be empty",
		[]*metricspb.Metric{cumulative(k1k2, timeseries(3, v1v2, double(3, 55)))},
		[]*metricspb.Metric{},
	}, {
		"Cumulative: round 4 - instance adjusted based on round 3",
		[]*metricspb.Metric{cumulative(k1k2, timeseries(4, v1v2, double(4, 72)))},
		[]*metricspb.Metric{cumulative(k1k2, timeseries(3, v1v2, double(4, 17)))},
	}}
	runScript(t, NewJobsMap(time.Duration(time.Minute)).get("job", "0"), script)
}

func Test_cumulativeDistribution(t *testing.T) {
	script := []*metricsAdjusterTest{{
		"CumulativeDist: round 1 - initial instance, adjusted should be empty",
		[]*metricspb.Metric{cumulativeDist(k1k2, timeseries(1, v1v2, dist(1, bounds0, []int64{4, 2, 3, 7})))},
		[]*metricspb.Metric{},
	}, {
		"CumulativeDist: round 2 - instance adjusted based on round 1",
		[]*metricspb.Metric{cumulativeDist(k1k2, timeseries(2, v1v2, dist(2, bounds0, []int64{6, 3, 4, 8})))},
		[]*metricspb.Metric{cumulativeDist(k1k2, timeseries(1, v1v2, dist(2, bounds0, []int64{2, 1, 1, 1})))},
	}, {
		"CumulativeDist: round 3 - instance reset (value less than previous value), adjusted should be empty",
		[]*metricspb.Metric{cumulativeDist(k1k2, timeseries(3, v1v2, dist(3, bounds0, []int64{5, 3, 2, 7})))},
		[]*metricspb.Metric{},
	}, {
		"CumulativeDist: round 4 - instance adjusted based on round 3",
		[]*metricspb.Metric{cumulativeDist(k1k2, timeseries(4, v1v2, dist(4, bounds0, []int64{7, 4, 2, 12})))},
		[]*metricspb.Metric{cumulativeDist(k1k2, timeseries(3, v1v2, dist(4, bounds0, []int64{2, 1, 0, 5})))},
	}}
	runScript(t, NewJobsMap(time.Duration(time.Minute)).get("job", "0"), script)
}

func Test_summary(t *testing.T) {
	script := []*metricsAdjusterTest{{
		"Summary: round 1 - initial instance, adjusted should be empty",
		[]*metricspb.Metric{summary(k1k2, timeseries(1, v1v2, summ(1, 10, 40, percent0, []float64{1, 5, 8})))},
		[]*metricspb.Metric{},
	}, {
		"Summary: round 2 - instance adjusted based on round 1",
		[]*metricspb.Metric{summary(k1k2, timeseries(2, v1v2, summ(2, 15, 70, percent0, []float64{7, 44, 9})))},
		[]*metricspb.Metric{summary(k1k2, timeseries(1, v1v2, summ(2, 5, 30, percent0, []float64{7, 44, 9})))},
	}, {
		"Summary: round 3 - instance reset (count less than previous), adjusted should be empty",
		[]*metricspb.Metric{summary(k1k2, timeseries(3, v1v2, summ(3, 12, 66, percent0, []float64{3, 22, 5})))},
		[]*metricspb.Metric{},
	}, {
		"Summary: round 4 - instance adjusted based on round 3",
		[]*metricspb.Metric{summary(k1k2, timeseries(4, v1v2, summ(4, 14, 96, percent0, []float64{9, 47, 8})))},
		[]*metricspb.Metric{summary(k1k2, timeseries(3, v1v2, summ(4, 2, 30, percent0, []float64{9, 47, 8})))},
	}}
	runScript(t, NewJobsMap(time.Duration(time.Minute)).get("job", "0"), script)
}

func Test_multiMetrics(t *testing.T) {
	script := []*metricsAdjusterTest{{
		"MultiMetrics: round 1 - combined round 1 of individual metrics",
		[]*metricspb.Metric{
			gauge(k1k2, timeseries(1, v1v2, double(1, 44))),
			gaugeDist(k1k2, timeseries(1, v1v2, dist(1, bounds0, []int64{4, 2, 3, 7}))),
			cumulative(k1k2, timeseries(1, v1v2, double(1, 44))),
			cumulativeDist(k1k2, timeseries(1, v1v2, dist(1, bounds0, []int64{4, 2, 3, 7}))),
			summary(k1k2, timeseries(1, v1v2, summ(1, 10, 40, percent0, []float64{1, 5, 8}))),
		},
		[]*metricspb.Metric{
			gauge(k1k2, timeseries(1, v1v2, double(1, 44))),
			gaugeDist(k1k2, timeseries(1, v1v2, dist(1, bounds0, []int64{4, 2, 3, 7}))),
		},
	}, {
		"MultiMetrics: round 2 - combined round 2 of individual metrics",
		[]*metricspb.Metric{
			gauge(k1k2, timeseries(2, v1v2, double(2, 66))),
			gaugeDist(k1k2, timeseries(2, v1v2, dist(2, bounds0, []int64{6, 5, 8, 11}))),
			cumulative(k1k2, timeseries(2, v1v2, double(2, 66))),
			cumulativeDist(k1k2, timeseries(2, v1v2, dist(2, bounds0, []int64{6, 3, 4, 8}))),
			summary(k1k2, timeseries(2, v1v2, summ(2, 15, 70, percent0, []float64{7, 44, 9}))),
		},
		[]*metricspb.Metric{
			gauge(k1k2, timeseries(2, v1v2, double(2, 66))),
			gaugeDist(k1k2, timeseries(2, v1v2, dist(2, bounds0, []int64{6, 5, 8, 11}))),
			cumulative(k1k2, timeseries(1, v1v2, double(2, 22))),
			cumulativeDist(k1k2, timeseries(1, v1v2, dist(2, bounds0, []int64{2, 1, 1, 1}))),
			summary(k1k2, timeseries(1, v1v2, summ(2, 5, 30, percent0, []float64{7, 44, 9}))),
		},
	}, {
		"MultiMetrics: round 3 - combined round 3 of individual metrics",
		[]*metricspb.Metric{
			gauge(k1k2, timeseries(3, v1v2, double(3, 55))),
			gaugeDist(k1k2, timeseries(3, v1v2, dist(3, bounds0, []int64{2, 0, 1, 5}))),
			cumulative(k1k2, timeseries(3, v1v2, double(3, 55))),
			cumulativeDist(k1k2, timeseries(3, v1v2, dist(3, bounds0, []int64{5, 3, 2, 7}))),
			summary(k1k2, timeseries(3, v1v2, summ(3, 12, 66, percent0, []float64{3, 22, 5}))),
		},
		[]*metricspb.Metric{
			gauge(k1k2, timeseries(3, v1v2, double(3, 55))),
			gaugeDist(k1k2, timeseries(3, v1v2, dist(3, bounds0, []int64{2, 0, 1, 5}))),
		},
	}, {
		"MultiMetrics: round 4 - combined round 4 of individual metrics",
		[]*metricspb.Metric{
			cumulative(k1k2, timeseries(4, v1v2, double(4, 72))),
			cumulativeDist(k1k2, timeseries(4, v1v2, dist(4, bounds0, []int64{7, 4, 2, 12}))),
			summary(k1k2, timeseries(4, v1v2, summ(4, 14, 96, percent0, []float64{9, 47, 8}))),
		},
		[]*metricspb.Metric{
			cumulative(k1k2, timeseries(3, v1v2, double(4, 17))),
			cumulativeDist(k1k2, timeseries(3, v1v2, dist(4, bounds0, []int64{2, 1, 0, 5}))),
			summary(k1k2, timeseries(3, v1v2, summ(4, 2, 30, percent0, []float64{9, 47, 8}))),
		},
	}}
	runScript(t, NewJobsMap(time.Duration(time.Minute)).get("job", "0"), script)
}

func Test_multiTimeseries(t *testing.T) {
	script := []*metricsAdjusterTest{{
		"MultiTimeseries: round 1 - initial first instance, adjusted should be empty",
		[]*metricspb.Metric{cumulative(k1k2, timeseries(1, v1v2, double(1, 44)))},
		[]*metricspb.Metric{},
	}, {
		"MultiTimeseries: round 2 - first instance adjusted based on round 1, initial second instance",
		[]*metricspb.Metric{cumulative(k1k2, timeseries(2, v1v2, double(2, 66)), timeseries(2, v10v20, double(2, 20)))},
		[]*metricspb.Metric{cumulative(k1k2, timeseries(1, v1v2, double(2, 22)))},
	}, {
		"MultiTimeseries: round 3 - first instance adjusted based on round 1, second based on round 2",
		[]*metricspb.Metric{cumulative(k1k2, timeseries(3, v1v2, double(3, 88)), timeseries(3, v10v20, double(3, 49)))},
		[]*metricspb.Metric{cumulative(k1k2, timeseries(1, v1v2, double(3, 44)), timeseries(2, v10v20, double(3, 29)))},
	}, {
		"MultiTimeseries: round 4 - first instance reset, second instance adjusted based on round 2, initial third instance",
		[]*metricspb.Metric{
			cumulative(k1k2, timeseries(4, v1v2, double(4, 87)), timeseries(4, v10v20, double(4, 57)), timeseries(4, v100v200, double(4, 10)))},
		[]*metricspb.Metric{
			cumulative(k1k2, timeseries(2, v10v20, double(4, 37)))},
	}, {
		"MultiTimeseries: round 5 - first instance adusted based on round 4, second on round 2, third on round 4",
		[]*metricspb.Metric{
			cumulative(k1k2, timeseries(5, v1v2, double(5, 90)), timeseries(5, v10v20, double(5, 65)), timeseries(5, v100v200, double(5, 22)))},
		[]*metricspb.Metric{
			cumulative(k1k2, timeseries(4, v1v2, double(5, 3)), timeseries(2, v10v20, double(5, 45)), timeseries(4, v100v200, double(5, 12)))},
	}}
	runScript(t, NewJobsMap(time.Duration(time.Minute)).get("job", "0"), script)
}

func Test_emptyLabels(t *testing.T) {
	script := []*metricsAdjusterTest{{
		"EmptyLabels: round 1 - initial instance, implicitly empty labels, adjusted should be empty",
		[]*metricspb.Metric{cumulative([]string{}, timeseries(1, []string{}, double(1, 44)))},
		[]*metricspb.Metric{},
	}, {
		"EmptyLabels: round 2 - instance adjusted based on round 1",
		[]*metricspb.Metric{cumulative([]string{}, timeseries(2, []string{}, double(2, 66)))},
		[]*metricspb.Metric{cumulative([]string{}, timeseries(1, []string{}, double(2, 22)))},
	}, {
		"EmptyLabels: round 3 - one explicitly empty label, instance adjusted based on round 1",
		[]*metricspb.Metric{cumulative(k1, timeseries(3, []string{""}, double(3, 77)))},
		[]*metricspb.Metric{cumulative(k1, timeseries(1, []string{""}, double(3, 33)))},
	}, {
		"EmptyLabels: round 4 - three explicitly empty labels, instance adjusted based on round 1",
		[]*metricspb.Metric{cumulative(k1k2k3, timeseries(3, []string{"", "", ""}, double(3, 88)))},
		[]*metricspb.Metric{cumulative(k1k2k3, timeseries(1, []string{"", "", ""}, double(3, 44)))},
	}}
	runScript(t, NewJobsMap(time.Duration(time.Minute)).get("job", "0"), script)
}

func Test_tsGC(t *testing.T) {
	script1 := []*metricsAdjusterTest{{
		"TsGC: round 1 - initial instances, adjusted should be empty",
		[]*metricspb.Metric{
			cumulative(k1k2, timeseries(1, v1v2, double(1, 44)), timeseries(1, v10v20, double(1, 20))),
			cumulativeDist(k1k2, timeseries(1, v1v2, dist(1, bounds0, []int64{4, 2, 3, 7})), timeseries(1, v10v20, dist(1, bounds0, []int64{40, 20, 30, 70}))),
		},
		[]*metricspb.Metric{},
	}}

	script2 := []*metricsAdjusterTest{{
		"TsGC: round 2 -  metrics first timeseries adjusted based on round 2, second timeseries not updated",
		[]*metricspb.Metric{
			cumulative(k1k2, timeseries(2, v1v2, double(2, 88))),
			cumulativeDist(k1k2, timeseries(2, v1v2, dist(2, bounds0, []int64{8, 7, 9, 14}))),
		},
		[]*metricspb.Metric{
			cumulative(k1k2, timeseries(1, v1v2, double(2, 44))),
			cumulativeDist(k1k2, timeseries(1, v1v2, dist(2, bounds0, []int64{4, 5, 6, 7}))),
		},
	}}

	script3 := []*metricsAdjusterTest{{
		"TsGC: round 3 - metrics first timeseries adjusted based on round 2, second timeseries empty due to timeseries gc()",
		[]*metricspb.Metric{
			cumulative(k1k2, timeseries(3, v1v2, double(3, 99)), timeseries(3, v10v20, double(3, 80))),
			cumulativeDist(k1k2, timeseries(3, v1v2, dist(3, bounds0, []int64{9, 8, 10, 15})), timeseries(3, v10v20, dist(3, bounds0, []int64{55, 66, 33, 77}))),
		},
		[]*metricspb.Metric{
			cumulative(k1k2, timeseries(1, v1v2, double(3, 55))),
			cumulativeDist(k1k2, timeseries(1, v1v2, dist(3, bounds0, []int64{5, 6, 7, 8}))),
		},
	}}

	jobsMap := NewJobsMap(time.Duration(time.Minute))

	// run round 1
	runScript(t, jobsMap.get("job", "0"), script1)
	// gc the tsmap, unmarking all entries
	jobsMap.get("job", "0").gc()
	// run round 2 - update metrics first timeseries only
	runScript(t, jobsMap.get("job", "0"), script2)
	// gc the tsmap, collecting umarked entries
	jobsMap.get("job", "0").gc()
	// run round 3 - verify that metrics second timeseries have been gc'd
	runScript(t, jobsMap.get("job", "0"), script3)
}

func Test_jobGC(t *testing.T) {
	job1Script1 := []*metricsAdjusterTest{{
		"JobGC: job 1, round 1 - initial instances, adjusted should be empty",
		[]*metricspb.Metric{
			cumulative(k1k2, timeseries(1, v1v2, double(1, 44)), timeseries(1, v10v20, double(1, 20))),
			cumulativeDist(k1k2, timeseries(1, v1v2, dist(1, bounds0, []int64{4, 2, 3, 7})), timeseries(1, v10v20, dist(1, bounds0, []int64{40, 20, 30, 70}))),
		},
		[]*metricspb.Metric{},
	}}

	job2Script1 := []*metricsAdjusterTest{{
		"JobGC: job2, round 1 - no metrics adjusted, just trigger gc",
		[]*metricspb.Metric{},
		[]*metricspb.Metric{},
	}}

	job1Script2 := []*metricsAdjusterTest{{
		"JobGC: job 1, round 2- metrics timeseries empty due to job-level gc",
		[]*metricspb.Metric{
			cumulative(k1k2, timeseries(4, v1v2, double(4, 99)), timeseries(4, v10v20, double(4, 80))),
			cumulativeDist(k1k2, timeseries(4, v1v2, dist(4, bounds0, []int64{9, 8, 10, 15})), timeseries(4, v10v20, dist(4, bounds0, []int64{55, 66, 33, 77}))),
		},
		[]*metricspb.Metric{},
	}}

	gcInterval := time.Duration(10 * time.Millisecond)
	jobsMap := NewJobsMap(gcInterval)

	// run job 1, round 1 - all entries marked
	runScript(t, jobsMap.get("job", "0"), job1Script1)
	// sleep longer than gcInterval to enable job gc in the next run
	time.Sleep(2 * gcInterval)
	// run job 2, round1 - trigger job gc, unmarking all entries
	runScript(t, jobsMap.get("job", "1"), job2Script1)
	// sleep longer than gcInterval to enable job gc in the next run
	time.Sleep(2 * gcInterval)
	// re-run job 2, round1 - trigger job gc, removing unmarked entries
	runScript(t, jobsMap.get("job", "1"), job2Script1)
	// run job 1, round 2 - verify that all job 1 timeseries have been gc'd
	runScript(t, jobsMap.get("job", "0"), job1Script2)
}

var (
	k1       = []string{"k1"}
	k1k2     = []string{"k1", "k2"}
	k1k2k3   = []string{"k1", "k2", "k3"}
	v1v2     = []string{"v1", "v2"}
	v10v20   = []string{"v10", "v20"}
	v100v200 = []string{"v100", "v200"}
	bounds0  = []float64{1, 2, 4}
	percent0 = []float64{10, 50, 90}
)

type metricsAdjusterTest struct {
	description string
	metrics     []*metricspb.Metric
	adjusted    []*metricspb.Metric
}

func runScript(t *testing.T, tsm *timeseriesMap, script []*metricsAdjusterTest) {
	l, _ := zap.NewProduction()
	defer l.Sync() // flushes buffer, if any
	ma := NewMetricsAdjuster(tsm, l.Sugar())

	for _, test := range script {
		adjusted := ma.AdjustMetrics(test.metrics)
		if !reflect.DeepEqual(test.adjusted, adjusted) {
			t.Errorf("Error: %v - expected: %v, actual %v", test.description, test.adjusted, adjusted)
			break
		}
	}
}

func gauge(keys []string, timeseries ...*metricspb.TimeSeries) *metricspb.Metric {
	return metric(metricspb.MetricDescriptor_GAUGE_DOUBLE, keys, timeseries)
}

func gaugeDist(keys []string, timeseries ...*metricspb.TimeSeries) *metricspb.Metric {
	return metric(metricspb.MetricDescriptor_GAUGE_DISTRIBUTION, keys, timeseries)
}

func cumulative(keys []string, timeseries ...*metricspb.TimeSeries) *metricspb.Metric {
	return metric(metricspb.MetricDescriptor_CUMULATIVE_DOUBLE, keys, timeseries)
}

func cumulativeDist(keys []string, timeseries ...*metricspb.TimeSeries) *metricspb.Metric {
	return metric(metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION, keys, timeseries)
}

func summary(keys []string, timeseries ...*metricspb.TimeSeries) *metricspb.Metric {
	return metric(metricspb.MetricDescriptor_SUMMARY, keys, timeseries)
}

func metric(ty metricspb.MetricDescriptor_Type, keys []string, timeseries []*metricspb.TimeSeries) *metricspb.Metric {
	return &metricspb.Metric{
		MetricDescriptor: metricDescriptor(ty, keys),
		Timeseries:       timeseries,
	}
}

func metricDescriptor(ty metricspb.MetricDescriptor_Type, keys []string) *metricspb.MetricDescriptor {
	return &metricspb.MetricDescriptor{
		Name:        ty.String(),
		Description: "description " + ty.String(),
		Unit:        "", // units not affected by adjuster
		Type:        ty,
		LabelKeys:   toKeys(keys),
	}
}

func timeseries(sts int64, vals []string, point *metricspb.Point) *metricspb.TimeSeries {
	return &metricspb.TimeSeries{
		StartTimestamp: toTS(sts),
		Points:         []*metricspb.Point{point},
		LabelValues:    toVals(vals),
	}
}

func double(ts int64, value float64) *metricspb.Point {
	return &metricspb.Point{Timestamp: toTS(ts), Value: &metricspb.Point_DoubleValue{DoubleValue: value}}
}

func dist(ts int64, bounds []float64, counts []int64) *metricspb.Point {
	var count int64
	var sum float64
	buckets := make([]*metricspb.DistributionValue_Bucket, len(counts))

	for i, bcount := range counts {
		count += bcount
		buckets[i] = &metricspb.DistributionValue_Bucket{Count: bcount}
		// create a sum based on lower bucket bounds
		// e.g. for bounds = {0.1, 0.2, 0.4} and counts = {2, 3, 7, 9)
		// sum = 0*2 + 0.1*3 + 0.2*7 + 0.4*9
		if i > 0 {
			sum += float64(bcount) * bounds[i-1]
		}
	}
	distrValue := &metricspb.DistributionValue{
		BucketOptions: &metricspb.DistributionValue_BucketOptions{
			Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
				Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
					Bounds: bounds,
				},
			},
		},
		Count:   count,
		Sum:     sum,
		Buckets: buckets,
		// SumOfSquaredDeviation:  // there's no way to compute this value from prometheus data
	}
	return &metricspb.Point{Timestamp: toTS(ts), Value: &metricspb.Point_DistributionValue{DistributionValue: distrValue}}
}

func summ(ts, count int64, sum float64, percent, vals []float64) *metricspb.Point {
	percentiles := make([]*metricspb.SummaryValue_Snapshot_ValueAtPercentile, len(percent))
	for i := 0; i < len(percent); i++ {
		percentiles[i] = &metricspb.SummaryValue_Snapshot_ValueAtPercentile{Percentile: percent[i], Value: vals[i]}
	}
	summaryValue := &metricspb.SummaryValue{
		Sum:   &wrappers.DoubleValue{Value: sum},
		Count: &wrappers.Int64Value{Value: count},
		Snapshot: &metricspb.SummaryValue_Snapshot{
			PercentileValues: percentiles,
		},
	}
	return &metricspb.Point{Timestamp: toTS(ts), Value: &metricspb.Point_SummaryValue{SummaryValue: summaryValue}}
}

func toKeys(keys []string) []*metricspb.LabelKey {
	res := make([]*metricspb.LabelKey, 0, len(keys))
	for _, key := range keys {
		res = append(res, &metricspb.LabelKey{Key: key, Description: "description: " + key})
	}
	return res
}

func toVals(vals []string) []*metricspb.LabelValue {
	res := make([]*metricspb.LabelValue, 0, len(vals))
	for _, val := range vals {
		res = append(res, &metricspb.LabelValue{Value: val, HasValue: true})
	}
	return res
}

func toTS(timeAtMs int64) *timestamp.Timestamp {
	secs, ns := timeAtMs/1e3, (timeAtMs%1e3)*1e6
	return &timestamp.Timestamp{
		Seconds: secs,
		Nanos:   int32(ns),
	}
}
