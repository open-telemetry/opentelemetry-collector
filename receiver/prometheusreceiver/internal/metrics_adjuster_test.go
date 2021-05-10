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
	"testing"
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	mtu "go.opentelemetry.io/collector/testutil/metricstestutil"
)

func Test_gauge(t *testing.T) {
	script := []*metricsAdjusterTest{{
		"Gauge: round 1 - gauge not adjusted",
		[]*metricspb.Metric{mtu.Gauge(g1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.Double(t1Ms, 44)))},
		[]*metricspb.Metric{mtu.Gauge(g1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.Double(t1Ms, 44)))},
		0,
	}, {
		"Gauge: round 2 - gauge not adjusted",
		[]*metricspb.Metric{mtu.Gauge(g1, k1k2, mtu.Timeseries(t2Ms, v1v2, mtu.Double(t2Ms, 66)))},
		[]*metricspb.Metric{mtu.Gauge(g1, k1k2, mtu.Timeseries(t2Ms, v1v2, mtu.Double(t2Ms, 66)))},
		0,
	}, {
		"Gauge: round 3 - value less than previous value - gauge is not adjusted",
		[]*metricspb.Metric{mtu.Gauge(g1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.Double(t3Ms, 55)))},
		[]*metricspb.Metric{mtu.Gauge(g1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.Double(t3Ms, 55)))},
		0,
	}}
	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_gaugeDistribution(t *testing.T) {
	script := []*metricsAdjusterTest{{
		"GaugeDist: round 1 - gauge distribution not adjusted",
		[]*metricspb.Metric{mtu.GaugeDist(gd1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.DistPt(t1Ms, bounds0, []int64{4, 2, 3, 7})))},
		[]*metricspb.Metric{mtu.GaugeDist(gd1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.DistPt(t1Ms, bounds0, []int64{4, 2, 3, 7})))},
		0,
	}, {
		"GaugeDist: round 2 - gauge distribution not adjusted",
		[]*metricspb.Metric{mtu.GaugeDist(gd1, k1k2, mtu.Timeseries(t2Ms, v1v2, mtu.DistPt(t2Ms, bounds0, []int64{6, 5, 8, 11})))},
		[]*metricspb.Metric{mtu.GaugeDist(gd1, k1k2, mtu.Timeseries(t2Ms, v1v2, mtu.DistPt(t2Ms, bounds0, []int64{6, 5, 8, 11})))},
		0,
	}, {
		"GaugeDist: round 3 - count/sum less than previous - gauge distribution not adjusted",
		[]*metricspb.Metric{mtu.GaugeDist(gd1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.DistPt(t3Ms, bounds0, []int64{2, 0, 1, 5})))},
		[]*metricspb.Metric{mtu.GaugeDist(gd1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.DistPt(t3Ms, bounds0, []int64{2, 0, 1, 5})))},
		0,
	}}
	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_cumulative(t *testing.T) {
	script := []*metricsAdjusterTest{{
		"Cumulative: round 1 - initial instance, start time is established",
		[]*metricspb.Metric{mtu.Cumulative(c1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.Double(t1Ms, 44)))},
		[]*metricspb.Metric{mtu.Cumulative(c1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.Double(t1Ms, 44)))},
		1,
	}, {
		"Cumulative: round 2 - instance adjusted based on round 1",
		[]*metricspb.Metric{mtu.Cumulative(c1, k1k2, mtu.Timeseries(t2Ms, v1v2, mtu.Double(t2Ms, 66)))},
		[]*metricspb.Metric{mtu.Cumulative(c1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.Double(t2Ms, 66)))},
		0,
	}, {
		"Cumulative: round 3 - instance reset (value less than previous value), start time is reset",
		[]*metricspb.Metric{mtu.Cumulative(c1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.Double(t3Ms, 55)))},
		[]*metricspb.Metric{mtu.Cumulative(c1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.Double(t3Ms, 55)))},
		1,
	}, {
		"Cumulative: round 4 - instance adjusted based on round 3",
		[]*metricspb.Metric{mtu.Cumulative(c1, k1k2, mtu.Timeseries(t4Ms, v1v2, mtu.Double(t4Ms, 72)))},
		[]*metricspb.Metric{mtu.Cumulative(c1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.Double(t4Ms, 72)))},
		0,
	}}
	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_cumulativeDistribution(t *testing.T) {
	script := []*metricsAdjusterTest{{
		"CumulativeDist: round 1 - initial instance, start time is established",
		[]*metricspb.Metric{mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.DistPt(t1Ms, bounds0, []int64{4, 2, 3, 7})))},
		[]*metricspb.Metric{mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.DistPt(t1Ms, bounds0, []int64{4, 2, 3, 7})))},
		1,
	}, {
		"CumulativeDist: round 2 - instance adjusted based on round 1",
		[]*metricspb.Metric{mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t2Ms, v1v2, mtu.DistPt(t2Ms, bounds0, []int64{6, 3, 4, 8})))},
		[]*metricspb.Metric{mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.DistPt(t2Ms, bounds0, []int64{6, 3, 4, 8})))},
		0,
	}, {
		"CumulativeDist: round 3 - instance reset (value less than previous value), start time is reset",
		[]*metricspb.Metric{mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.DistPt(t3Ms, bounds0, []int64{5, 3, 2, 7})))},
		[]*metricspb.Metric{mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.DistPt(t3Ms, bounds0, []int64{5, 3, 2, 7})))},
		1,
	}, {
		"CumulativeDist: round 4 - instance adjusted based on round 3",
		[]*metricspb.Metric{mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t4Ms, v1v2, mtu.DistPt(t4Ms, bounds0, []int64{7, 4, 2, 12})))},
		[]*metricspb.Metric{mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.DistPt(t4Ms, bounds0, []int64{7, 4, 2, 12})))},
		0,
	}}
	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_summary(t *testing.T) {
	script := []*metricsAdjusterTest{{
		"Summary: round 1 - initial instance, start time is established",
		[]*metricspb.Metric{mtu.Summary(s1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.SummPt(t1Ms, 10, 40, percent0, []float64{1, 5, 8})))},
		[]*metricspb.Metric{mtu.Summary(s1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.SummPt(t1Ms, 10, 40, percent0, []float64{1, 5, 8})))},
		1,
	}, {
		"Summary: round 2 - instance adjusted based on round 1",
		[]*metricspb.Metric{mtu.Summary(s1, k1k2, mtu.Timeseries(t2Ms, v1v2, mtu.SummPt(t2Ms, 15, 70, percent0, []float64{7, 44, 9})))},
		[]*metricspb.Metric{mtu.Summary(s1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.SummPt(t2Ms, 15, 70, percent0, []float64{7, 44, 9})))},
		0,
	}, {
		"Summary: round 3 - instance reset (count less than previous), start time is reset",
		[]*metricspb.Metric{mtu.Summary(s1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.SummPt(t3Ms, 12, 66, percent0, []float64{3, 22, 5})))},
		[]*metricspb.Metric{mtu.Summary(s1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.SummPt(t3Ms, 12, 66, percent0, []float64{3, 22, 5})))},
		1,
	}, {
		"Summary: round 4 - instance adjusted based on round 3",
		[]*metricspb.Metric{mtu.Summary(s1, k1k2, mtu.Timeseries(t4Ms, v1v2, mtu.SummPt(t4Ms, 14, 96, percent0, []float64{9, 47, 8})))},
		[]*metricspb.Metric{mtu.Summary(s1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.SummPt(t4Ms, 14, 96, percent0, []float64{9, 47, 8})))},
		0,
	}}
	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_multiMetrics(t *testing.T) {
	script := []*metricsAdjusterTest{{
		"MultiMetrics: round 1 - combined round 1 of individual metrics",
		[]*metricspb.Metric{
			mtu.Gauge(g1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.Double(t1Ms, 44))),
			mtu.GaugeDist(gd1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.DistPt(t1Ms, bounds0, []int64{4, 2, 3, 7}))),
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.Double(t1Ms, 44))),
			mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.DistPt(t1Ms, bounds0, []int64{4, 2, 3, 7}))),
			mtu.Summary(s1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.SummPt(t1Ms, 10, 40, percent0, []float64{1, 5, 8}))),
		},
		[]*metricspb.Metric{
			mtu.Gauge(g1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.Double(t1Ms, 44))),
			mtu.GaugeDist(gd1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.DistPt(t1Ms, bounds0, []int64{4, 2, 3, 7}))),
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.Double(t1Ms, 44))),
			mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.DistPt(t1Ms, bounds0, []int64{4, 2, 3, 7}))),
			mtu.Summary(s1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.SummPt(t1Ms, 10, 40, percent0, []float64{1, 5, 8}))),
		},
		3,
	}, {
		"MultiMetrics: round 2 - combined round 2 of individual metrics",
		[]*metricspb.Metric{
			mtu.Gauge(g1, k1k2, mtu.Timeseries(t2Ms, v1v2, mtu.Double(t2Ms, 66))),
			mtu.GaugeDist(gd1, k1k2, mtu.Timeseries(t2Ms, v1v2, mtu.DistPt(t2Ms, bounds0, []int64{6, 5, 8, 11}))),
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t2Ms, v1v2, mtu.Double(t2Ms, 66))),
			mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t2Ms, v1v2, mtu.DistPt(t2Ms, bounds0, []int64{6, 3, 4, 8}))),
			mtu.Summary(s1, k1k2, mtu.Timeseries(t2Ms, v1v2, mtu.SummPt(t2Ms, 15, 70, percent0, []float64{7, 44, 9}))),
		},
		[]*metricspb.Metric{
			mtu.Gauge(g1, k1k2, mtu.Timeseries(t2Ms, v1v2, mtu.Double(t2Ms, 66))),
			mtu.GaugeDist(gd1, k1k2, mtu.Timeseries(t2Ms, v1v2, mtu.DistPt(t2Ms, bounds0, []int64{6, 5, 8, 11}))),
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.Double(t2Ms, 66))),
			mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.DistPt(t2Ms, bounds0, []int64{6, 3, 4, 8}))),
			mtu.Summary(s1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.SummPt(t2Ms, 15, 70, percent0, []float64{7, 44, 9}))),
		},
		0,
	}, {
		"MultiMetrics: round 3 - combined round 3 of individual metrics",
		[]*metricspb.Metric{
			mtu.Gauge(g1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.Double(t3Ms, 55))),
			mtu.GaugeDist(gd1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.DistPt(t3Ms, bounds0, []int64{2, 0, 1, 5}))),
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.Double(t3Ms, 55))),
			mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.DistPt(t3Ms, bounds0, []int64{5, 3, 2, 7}))),
			mtu.Summary(s1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.SummPt(t3Ms, 12, 66, percent0, []float64{3, 22, 5}))),
		},
		[]*metricspb.Metric{
			mtu.Gauge(g1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.Double(t3Ms, 55))),
			mtu.GaugeDist(gd1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.DistPt(t3Ms, bounds0, []int64{2, 0, 1, 5}))),
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.Double(t3Ms, 55))),
			mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.DistPt(t3Ms, bounds0, []int64{5, 3, 2, 7}))),
			mtu.Summary(s1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.SummPt(t3Ms, 12, 66, percent0, []float64{3, 22, 5}))),
		},
		3,
	}, {
		"MultiMetrics: round 4 - combined round 4 of individual metrics",
		[]*metricspb.Metric{
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t4Ms, v1v2, mtu.Double(t4Ms, 72))),
			mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t4Ms, v1v2, mtu.DistPt(t4Ms, bounds0, []int64{7, 4, 2, 12}))),
			mtu.Summary(s1, k1k2, mtu.Timeseries(t4Ms, v1v2, mtu.SummPt(t4Ms, 14, 96, percent0, []float64{9, 47, 8}))),
		},
		[]*metricspb.Metric{
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.Double(t4Ms, 72))),
			mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.DistPt(t4Ms, bounds0, []int64{7, 4, 2, 12}))),
			mtu.Summary(s1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.SummPt(t4Ms, 14, 96, percent0, []float64{9, 47, 8}))),
		},
		0,
	}}
	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_multiTimeseries(t *testing.T) {
	script := []*metricsAdjusterTest{{
		"MultiTimeseries: round 1 - initial first instance, start time is established",
		[]*metricspb.Metric{mtu.Cumulative(c1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.Double(t1Ms, 44)))},
		[]*metricspb.Metric{mtu.Cumulative(c1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.Double(t1Ms, 44)))},
		1,
	}, {
		"MultiTimeseries: round 2 - first instance adjusted based on round 1, initial second instance",
		[]*metricspb.Metric{mtu.Cumulative(c1, k1k2, mtu.Timeseries(t2Ms, v1v2, mtu.Double(t2Ms, 66)), mtu.Timeseries(t2Ms, v10v20, mtu.Double(t2Ms, 20)))},
		[]*metricspb.Metric{mtu.Cumulative(c1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.Double(t2Ms, 66)), mtu.Timeseries(t2Ms, v10v20, mtu.Double(t2Ms, 20)))},
		1,
	}, {
		"MultiTimeseries: round 3 - first instance adjusted based on round 1, second based on round 2",
		[]*metricspb.Metric{mtu.Cumulative(c1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.Double(t3Ms, 88)), mtu.Timeseries(t3Ms, v10v20, mtu.Double(t3Ms, 49)))},
		[]*metricspb.Metric{mtu.Cumulative(c1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.Double(t3Ms, 88)), mtu.Timeseries(t2Ms, v10v20, mtu.Double(t3Ms, 49)))},
		0,
	}, {
		"MultiTimeseries: round 4 - first instance reset, second instance adjusted based on round 2, initial third instance",
		[]*metricspb.Metric{
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t4Ms, v1v2, mtu.Double(t4Ms, 87)), mtu.Timeseries(t4Ms, v10v20, mtu.Double(t4Ms, 57)), mtu.Timeseries(t4Ms, v100v200, mtu.Double(t4Ms, 10)))},
		[]*metricspb.Metric{
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t4Ms, v1v2, mtu.Double(t4Ms, 87)), mtu.Timeseries(t2Ms, v10v20, mtu.Double(t4Ms, 57)), mtu.Timeseries(t4Ms, v100v200, mtu.Double(t4Ms, 10)))},
		2,
	}, {
		"MultiTimeseries: round 5 - first instance adjusted based on round 4, second on round 2, third on round 4",
		[]*metricspb.Metric{
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t5Ms, v1v2, mtu.Double(t5Ms, 90)), mtu.Timeseries(t5Ms, v10v20, mtu.Double(t5Ms, 65)), mtu.Timeseries(t5Ms, v100v200, mtu.Double(t5Ms, 22)))},
		[]*metricspb.Metric{
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t4Ms, v1v2, mtu.Double(t5Ms, 90)), mtu.Timeseries(t2Ms, v10v20, mtu.Double(t5Ms, 65)), mtu.Timeseries(t4Ms, v100v200, mtu.Double(t5Ms, 22)))},
		0,
	}}
	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_emptyLabels(t *testing.T) {
	script := []*metricsAdjusterTest{{
		"EmptyLabels: round 1 - initial instance, implicitly empty labels, start time is established",
		[]*metricspb.Metric{mtu.Cumulative(c1, []string{}, mtu.Timeseries(t1Ms, []string{}, mtu.Double(t1Ms, 44)))},
		[]*metricspb.Metric{mtu.Cumulative(c1, []string{}, mtu.Timeseries(t1Ms, []string{}, mtu.Double(t1Ms, 44)))},
		1,
	}, {
		"EmptyLabels: round 2 - instance adjusted based on round 1",
		[]*metricspb.Metric{mtu.Cumulative(c1, []string{}, mtu.Timeseries(t2Ms, []string{}, mtu.Double(t2Ms, 66)))},
		[]*metricspb.Metric{mtu.Cumulative(c1, []string{}, mtu.Timeseries(t1Ms, []string{}, mtu.Double(t2Ms, 66)))},
		0,
	}, {
		"EmptyLabels: round 3 - one explicitly empty label, instance adjusted based on round 1",
		[]*metricspb.Metric{mtu.Cumulative(c1, k1, mtu.Timeseries(t3Ms, []string{""}, mtu.Double(t3Ms, 77)))},
		[]*metricspb.Metric{mtu.Cumulative(c1, k1, mtu.Timeseries(t1Ms, []string{""}, mtu.Double(t3Ms, 77)))},
		0,
	}, {
		"EmptyLabels: round 4 - three explicitly empty labels, instance adjusted based on round 1",
		[]*metricspb.Metric{mtu.Cumulative(c1, k1k2k3, mtu.Timeseries(t3Ms, []string{"", "", ""}, mtu.Double(t3Ms, 88)))},
		[]*metricspb.Metric{mtu.Cumulative(c1, k1k2k3, mtu.Timeseries(t1Ms, []string{"", "", ""}, mtu.Double(t3Ms, 88)))},
		0,
	}}
	runScript(t, NewJobsMap(time.Minute).get("job", "0"), script)
}

func Test_tsGC(t *testing.T) {
	script1 := []*metricsAdjusterTest{{
		"TsGC: round 1 - initial instances, start time is established",
		[]*metricspb.Metric{
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.Double(t1Ms, 44)), mtu.Timeseries(t1Ms, v10v20, mtu.Double(t1Ms, 20))),
			mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.DistPt(t1Ms, bounds0, []int64{4, 2, 3, 7})), mtu.Timeseries(t1Ms, v10v20, mtu.DistPt(t1Ms, bounds0, []int64{40, 20, 30, 70}))),
		},
		[]*metricspb.Metric{
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.Double(t1Ms, 44)), mtu.Timeseries(t1Ms, v10v20, mtu.Double(t1Ms, 20))),
			mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.DistPt(t1Ms, bounds0, []int64{4, 2, 3, 7})), mtu.Timeseries(t1Ms, v10v20, mtu.DistPt(t1Ms, bounds0, []int64{40, 20, 30, 70}))),
		},
		4,
	}}

	script2 := []*metricsAdjusterTest{{
		"TsGC: round 2 - metrics first timeseries adjusted based on round 2, second timeseries not updated",
		[]*metricspb.Metric{
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t2Ms, v1v2, mtu.Double(t2Ms, 88))),
			mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t2Ms, v1v2, mtu.DistPt(t2Ms, bounds0, []int64{8, 7, 9, 14}))),
		},
		[]*metricspb.Metric{
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.Double(t2Ms, 88))),
			mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.DistPt(t2Ms, bounds0, []int64{8, 7, 9, 14}))),
		},
		0,
	}}

	script3 := []*metricsAdjusterTest{{
		"TsGC: round 3 - metrics first timeseries adjusted based on round 2, second timeseries empty due to timeseries gc()",
		[]*metricspb.Metric{
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.Double(t3Ms, 99)), mtu.Timeseries(t3Ms, v10v20, mtu.Double(t3Ms, 80))),
			mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t3Ms, v1v2, mtu.DistPt(t3Ms, bounds0, []int64{9, 8, 10, 15})), mtu.Timeseries(t3Ms, v10v20, mtu.DistPt(t3Ms, bounds0, []int64{55, 66, 33, 77}))),
		},
		[]*metricspb.Metric{
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.Double(t3Ms, 99)), mtu.Timeseries(t3Ms, v10v20, mtu.Double(t3Ms, 80))),
			mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.DistPt(t3Ms, bounds0, []int64{9, 8, 10, 15})), mtu.Timeseries(t3Ms, v10v20, mtu.DistPt(t3Ms, bounds0, []int64{55, 66, 33, 77}))),
		},
		2,
	}}

	jobsMap := NewJobsMap(time.Minute)

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
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.Double(t1Ms, 44)), mtu.Timeseries(t1Ms, v10v20, mtu.Double(t1Ms, 20))),
			mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.DistPt(t1Ms, bounds0, []int64{4, 2, 3, 7})), mtu.Timeseries(t1Ms, v10v20, mtu.DistPt(t1Ms, bounds0, []int64{40, 20, 30, 70}))),
		},
		[]*metricspb.Metric{
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.Double(t1Ms, 44)), mtu.Timeseries(t1Ms, v10v20, mtu.Double(t1Ms, 20))),
			mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t1Ms, v1v2, mtu.DistPt(t1Ms, bounds0, []int64{4, 2, 3, 7})), mtu.Timeseries(t1Ms, v10v20, mtu.DistPt(t1Ms, bounds0, []int64{40, 20, 30, 70}))),
		},
		4,
	}}

	job2Script1 := []*metricsAdjusterTest{{
		"JobGC: job2, round 1 - no metrics adjusted, just trigger gc",
		[]*metricspb.Metric{},
		[]*metricspb.Metric{},
		0,
	}}

	job1Script2 := []*metricsAdjusterTest{{
		"JobGC: job 1, round 2 - metrics timeseries empty due to job-level gc",
		[]*metricspb.Metric{
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t4Ms, v1v2, mtu.Double(t4Ms, 99)), mtu.Timeseries(t4Ms, v10v20, mtu.Double(t4Ms, 80))),
			mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t4Ms, v1v2, mtu.DistPt(t4Ms, bounds0, []int64{9, 8, 10, 15})), mtu.Timeseries(t4Ms, v10v20, mtu.DistPt(t4Ms, bounds0, []int64{55, 66, 33, 77}))),
		},
		[]*metricspb.Metric{
			mtu.Cumulative(c1, k1k2, mtu.Timeseries(t4Ms, v1v2, mtu.Double(t4Ms, 99)), mtu.Timeseries(t4Ms, v10v20, mtu.Double(t4Ms, 80))),
			mtu.CumulativeDist(cd1, k1k2, mtu.Timeseries(t4Ms, v1v2, mtu.DistPt(t4Ms, bounds0, []int64{9, 8, 10, 15})), mtu.Timeseries(t4Ms, v10v20, mtu.DistPt(t4Ms, bounds0, []int64{55, 66, 33, 77}))),
		},
		4,
	}}

	gcInterval := 10 * time.Millisecond
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
	// ensure that at least one jobsMap.gc() completed
	jobsMap.gc()
	// run job 1, round 2 - verify that all job 1 timeseries have been gc'd
	runScript(t, jobsMap.get("job", "0"), job1Script2)
}

var (
	g1       = "gauge1"
	gd1      = "gaugedist1"
	c1       = "cumulative1"
	cd1      = "cumulativedist1"
	s1       = "summary1"
	k1       = []string{"k1"}
	k1k2     = []string{"k1", "k2"}
	k1k2k3   = []string{"k1", "k2", "k3"}
	v1v2     = []string{"v1", "v2"}
	v10v20   = []string{"v10", "v20"}
	v100v200 = []string{"v100", "v200"}
	bounds0  = []float64{1, 2, 4}
	percent0 = []float64{10, 50, 90}
	t1Ms     = time.Unix(0, 1000000)
	t2Ms     = time.Unix(0, 2000000)
	t3Ms     = time.Unix(0, 3000000)
	t4Ms     = time.Unix(0, 5000000)
	t5Ms     = time.Unix(0, 5000000)
)

type metricsAdjusterTest struct {
	description string
	metrics     []*metricspb.Metric
	adjusted    []*metricspb.Metric
	resets      int
}

func runScript(t *testing.T, tsm *timeseriesMap, script []*metricsAdjusterTest) {
	l := zap.NewNop()
	t.Cleanup(func() { require.NoError(t, l.Sync()) }) // flushes buffer, if any
	ma := NewMetricsAdjuster(tsm, l)

	for _, test := range script {
		expectedResets := test.resets
		adjusted, resets := ma.AdjustMetrics(test.metrics)
		assert.EqualValuesf(t, test.adjusted, adjusted, "Test: %v - expected: %v, actual: %v", test.description, test.adjusted, adjusted)
		assert.Equalf(t, expectedResets, resets, "Test: %v", test.description)
	}
}
