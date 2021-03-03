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

package metricstestutil

import (
	"testing"
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestHelpers(t *testing.T) {
	op1 := "op1"
	op2 := "op2"
	k1k2 := []string{"k1", "k2"}
	v1v2 := []string{"v1", "v2"}
	v10v20 := []string{"v10", "v20"}
	bounds0 := []float64{1}
	percent0 := []float64{10}
	t1Ms := time.Unix(0, 1000000)
	t3Ms := time.Unix(0, 3000000)
	t5Ms := time.Unix(0, 5000000)

	k1k2Labels := []*metricspb.LabelKey{
		{Key: "k1", Description: "description: k1"},
		{Key: "k2", Description: "description: k2"},
	}

	v1v2Values := []*metricspb.LabelValue{
		{Value: "v1", HasValue: true},
		{Value: "v2", HasValue: true},
	}

	v10v20Values := []*metricspb.LabelValue{
		{Value: "v10", HasValue: true},
		{Value: "v20", HasValue: true},
	}

	ts1Ms := timestamppb.New(t1Ms)
	ts3Ms := timestamppb.New(t3Ms)
	ts5Ms := timestamppb.New(t5Ms)

	d44 := &metricspb.Point_DoubleValue{DoubleValue: 44}
	d65 := &metricspb.Point_DoubleValue{DoubleValue: 65}
	d90 := &metricspb.Point_DoubleValue{DoubleValue: 90}

	dist := &metricspb.Point_DistributionValue{
		DistributionValue: &metricspb.DistributionValue{
			BucketOptions: &metricspb.DistributionValue_BucketOptions{
				Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
					Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
						Bounds: []float64{1},
					},
				},
			},
			Count:   2,
			Sum:     0,
			Buckets: []*metricspb.DistributionValue_Bucket{{Count: 2}},
		},
	}

	summ := &metricspb.Point_SummaryValue{
		SummaryValue: &metricspb.SummaryValue{
			Sum:   wrapperspb.Double(40),
			Count: wrapperspb.Int64(10),
			Snapshot: &metricspb.SummaryValue_Snapshot{
				PercentileValues: []*metricspb.SummaryValue_Snapshot_ValueAtPercentile{
					{Percentile: 10, Value: 1},
				},
			},
		},
	}

	got := []*metricspb.Metric{
		Gauge(op1, k1k2, Timeseries(t1Ms, v1v2, Double(t1Ms, 44))),
		GaugeDist(op2, k1k2, Timeseries(t3Ms, v1v2, DistPt(t1Ms, bounds0, []int64{2}))),
		Cumulative(op1, k1k2, Timeseries(t5Ms, v1v2, Double(t5Ms, 90)), Timeseries(t5Ms, v10v20, Double(t5Ms, 65))),
		CumulativeDist(op2, k1k2, Timeseries(t1Ms, v1v2, DistPt(t1Ms, bounds0, []int64{2}))),
		Summary(op1, k1k2, Timeseries(t1Ms, v1v2, SummPt(t1Ms, 10, 40, percent0, []float64{1, 5}))),
	}

	want := []*metricspb.Metric{
		{
			MetricDescriptor: &metricspb.MetricDescriptor{
				Name:        op1,
				Description: "metrics description",
				Type:        metricspb.MetricDescriptor_GAUGE_DOUBLE,
				LabelKeys:   k1k2Labels,
			},
			Timeseries: []*metricspb.TimeSeries{
				{
					StartTimestamp: ts1Ms,
					LabelValues:    v1v2Values,
					Points:         []*metricspb.Point{{Timestamp: ts1Ms, Value: d44}},
				},
			},
		},
		{
			MetricDescriptor: &metricspb.MetricDescriptor{
				Name:        op2,
				Description: "metrics description",
				Type:        metricspb.MetricDescriptor_GAUGE_DISTRIBUTION,
				LabelKeys:   k1k2Labels,
			},
			Timeseries: []*metricspb.TimeSeries{
				{
					StartTimestamp: ts3Ms,
					LabelValues:    v1v2Values,
					Points:         []*metricspb.Point{{Timestamp: ts1Ms, Value: dist}},
				},
			},
		},
		{
			MetricDescriptor: &metricspb.MetricDescriptor{
				Name:        op1,
				Description: "metrics description",
				Type:        metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
				LabelKeys:   k1k2Labels,
			},
			Timeseries: []*metricspb.TimeSeries{
				{
					StartTimestamp: ts5Ms,
					LabelValues:    v1v2Values,
					Points:         []*metricspb.Point{{Timestamp: ts5Ms, Value: d90}},
				},
				{
					StartTimestamp: ts5Ms,
					LabelValues:    v10v20Values,
					Points:         []*metricspb.Point{{Timestamp: ts5Ms, Value: d65}},
				},
			},
		},
		{
			MetricDescriptor: &metricspb.MetricDescriptor{
				Name:        op2,
				Description: "metrics description",
				Type:        metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
				LabelKeys:   k1k2Labels,
			},
			Timeseries: []*metricspb.TimeSeries{
				{
					StartTimestamp: ts1Ms,
					LabelValues:    v1v2Values,
					Points:         []*metricspb.Point{{Timestamp: ts1Ms, Value: dist}},
				},
			},
		},
		{
			MetricDescriptor: &metricspb.MetricDescriptor{
				Name:        op1,
				Description: "metrics description",
				Type:        metricspb.MetricDescriptor_SUMMARY,
				LabelKeys:   k1k2Labels,
			},
			Timeseries: []*metricspb.TimeSeries{
				{
					StartTimestamp: ts1Ms,
					LabelValues:    v1v2Values,
					Points:         []*metricspb.Point{{Timestamp: ts1Ms, Value: summ}},
				},
			},
		},
	}
	assert.Equal(t, want, got)
}
