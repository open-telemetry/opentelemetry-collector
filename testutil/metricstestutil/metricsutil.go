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
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Gauge creates a gauge metric.
func Gauge(name string, keys []string, timeseries ...*metricspb.TimeSeries) *metricspb.Metric {
	return metric(metricspb.MetricDescriptor_GAUGE_DOUBLE, name, keys, timeseries)
}

// GaugeInt creates a gauge metric of type int64.
func GaugeInt(name string, keys []string, timeseries ...*metricspb.TimeSeries) *metricspb.Metric {
	return metric(metricspb.MetricDescriptor_GAUGE_INT64, name, keys, timeseries)
}

// GaugeDist creates a gauge distribution metric.
func GaugeDist(name string, keys []string, timeseries ...*metricspb.TimeSeries) *metricspb.Metric {
	return metric(metricspb.MetricDescriptor_GAUGE_DISTRIBUTION, name, keys, timeseries)
}

// Cumulative creates a cumulative metric.
func Cumulative(name string, keys []string, timeseries ...*metricspb.TimeSeries) *metricspb.Metric {
	return metric(metricspb.MetricDescriptor_CUMULATIVE_DOUBLE, name, keys, timeseries)
}

// CumulativeInt creates a cumulative metric of type int64.
func CumulativeInt(name string, keys []string, timeseries ...*metricspb.TimeSeries) *metricspb.Metric {
	return metric(metricspb.MetricDescriptor_CUMULATIVE_INT64, name, keys, timeseries)
}

// CumulativeDist creates a cumulative distribution metric.
func CumulativeDist(name string, keys []string, timeseries ...*metricspb.TimeSeries) *metricspb.Metric {
	return metric(metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION, name, keys, timeseries)
}

// Summary creates a summary metric.
func Summary(name string, keys []string, timeseries ...*metricspb.TimeSeries) *metricspb.Metric {
	return metric(metricspb.MetricDescriptor_SUMMARY, name, keys, timeseries)
}

// Timeseries creates a timeseries. It takes the start time stamp, a sequence of label values (associated
// with the label keys in the overall metric), and the value of the timeseries.
func Timeseries(sts time.Time, vals []string, point *metricspb.Point) *metricspb.TimeSeries {
	return &metricspb.TimeSeries{
		StartTimestamp: timestamppb.New(sts),
		Points:         []*metricspb.Point{point},
		LabelValues:    toVals(vals),
	}
}

// Double creates a double point.
func Double(ts time.Time, value float64) *metricspb.Point {
	return &metricspb.Point{Timestamp: timestamppb.New(ts), Value: &metricspb.Point_DoubleValue{DoubleValue: value}}
}

// DistPt creates a distribution point. It takes the time stamp, the bucket boundaries for the distribution, and
// the and counts for the individual buckets as input.
func DistPt(ts time.Time, bounds []float64, counts []int64) *metricspb.Point {
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
		// There's no way to compute SumOfSquaredDeviation from prometheus data
	}
	return &metricspb.Point{Timestamp: timestamppb.New(ts), Value: &metricspb.Point_DistributionValue{DistributionValue: distrValue}}
}

// SummPt creates a summary point.
func SummPt(ts time.Time, count int64, sum float64, percent, vals []float64) *metricspb.Point {
	percentiles := make([]*metricspb.SummaryValue_Snapshot_ValueAtPercentile, len(percent))
	for i := 0; i < len(percent); i++ {
		percentiles[i] = &metricspb.SummaryValue_Snapshot_ValueAtPercentile{Percentile: percent[i], Value: vals[i]}
	}
	summaryValue := &metricspb.SummaryValue{
		Sum:   wrapperspb.Double(sum),
		Count: wrapperspb.Int64(count),
		Snapshot: &metricspb.SummaryValue_Snapshot{
			PercentileValues: percentiles,
		},
	}
	return &metricspb.Point{Timestamp: timestamppb.New(ts), Value: &metricspb.Point_SummaryValue{SummaryValue: summaryValue}}
}

func metric(ty metricspb.MetricDescriptor_Type, name string, keys []string, timeseries []*metricspb.TimeSeries) *metricspb.Metric {
	return &metricspb.Metric{
		MetricDescriptor: &metricspb.MetricDescriptor{
			Name:        name,
			Description: "metrics description",
			Unit:        "",
			Type:        ty,
			LabelKeys:   toKeys(keys),
		},
		Timeseries: timeseries,
	}
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
