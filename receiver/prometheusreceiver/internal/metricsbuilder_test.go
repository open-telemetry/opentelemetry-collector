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

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/prometheus/prometheus/scrape"

	"github.com/open-telemetry/opentelemetry-service/exporter/exportertest"
)

func Test_isUsefulLabel(t *testing.T) {
	type args struct {
		mType    metricspb.MetricDescriptor_Type
		labelKey string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"metricName", args{metricspb.MetricDescriptor_GAUGE_DOUBLE, model.MetricNameLabel}, false},
		{"instance", args{metricspb.MetricDescriptor_GAUGE_DOUBLE, model.InstanceLabel}, false},
		{"scheme", args{metricspb.MetricDescriptor_GAUGE_DOUBLE, model.SchemeLabel}, false},
		{"metricPath", args{metricspb.MetricDescriptor_GAUGE_DOUBLE, model.MetricsPathLabel}, false},
		{"job", args{metricspb.MetricDescriptor_GAUGE_DOUBLE, model.JobLabel}, false},
		{"bucket", args{metricspb.MetricDescriptor_GAUGE_DOUBLE, model.BucketLabel}, true},
		{"bucketForGaugeDistribution", args{metricspb.MetricDescriptor_GAUGE_DISTRIBUTION, model.BucketLabel}, false},
		{"bucketForCumulativeDistribution", args{metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION, model.BucketLabel}, false},
		{"Quantile", args{metricspb.MetricDescriptor_GAUGE_DOUBLE, model.QuantileLabel}, true},
		{"QuantileForSummay", args{metricspb.MetricDescriptor_SUMMARY, model.QuantileLabel}, false},
		{"other", args{metricspb.MetricDescriptor_GAUGE_DOUBLE, "other"}, true},
		{"empty", args{metricspb.MetricDescriptor_GAUGE_DOUBLE, ""}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUsefulLabel(tt.args.mType, tt.args.labelKey); got != tt.want {
				t.Errorf("isUsefulLabel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dpgSignature(t *testing.T) {
	knownLabelKeys := map[string]int{"a": 0, "b": 1}

	tests := []struct {
		name string
		ls   labels.Labels
		want string
	}{
		{"1st label", labels.FromStrings("a", "va"), `[]string{"va", ""}`},
		{"2nd label", labels.FromStrings("b", "vb"), `[]string{"", "vb"}`},
		{"two labels", labels.FromStrings("a", "va", "b", "vb"), `[]string{"va", "vb"}`},
		{"extra label", labels.FromStrings("a", "va", "b", "vb", "x", "xa"), `[]string{"va", "vb"}`},
		{"different order", labels.FromStrings("b", "vb", "a", "va"), `[]string{"va", "vb"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dpgSignature(knownLabelKeys, tt.ls); got != tt.want {
				t.Errorf("dpgSignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_normalizeMetricName(t *testing.T) {
	tests := []struct {
		name  string
		mname string
		want  string
	}{
		{"normal", "normal", "normal"},
		{"count", "foo_count", "foo"},
		{"bucket", "foo_bucket", "foo"},
		{"sum", "foo_sum", "foo"},
		{"no_prefix", "_sum", "_sum"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeMetricName(tt.mname); got != tt.want {
				t.Errorf("normalizeMetricName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getBoundary(t *testing.T) {
	ls := labels.FromStrings("le", "100.0", "foo", "bar", "quantile", "0.5")
	ls2 := labels.FromStrings("foo", "bar")
	ls3 := labels.FromStrings("le", "xyz", "foo", "bar", "quantile", "0.5")
	type args struct {
		metricType metricspb.MetricDescriptor_Type
		labels     labels.Labels
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{"histogram", args{metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION, ls}, 100.0, false},
		{"gaugehistogram", args{metricspb.MetricDescriptor_GAUGE_DISTRIBUTION, ls}, 100.0, false},
		{"gaugehistogram_no_label", args{metricspb.MetricDescriptor_GAUGE_DISTRIBUTION, ls2}, 0, true},
		{"gaugehistogram_bad_value", args{metricspb.MetricDescriptor_GAUGE_DISTRIBUTION, ls3}, 0, true},
		{"summary", args{metricspb.MetricDescriptor_SUMMARY, ls}, 0.5, false},
		{"otherType", args{metricspb.MetricDescriptor_GAUGE_DOUBLE, ls}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getBoundary(tt.args.metricType, tt.args.labels)
			if (err != nil) != tt.wantErr {
				t.Errorf("getBoundary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getBoundary() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convToOCAMetricType(t *testing.T) {
	tests := []struct {
		name       string
		metricType textparse.MetricType
		want       metricspb.MetricDescriptor_Type
	}{
		{"counter", textparse.MetricTypeCounter, metricspb.MetricDescriptor_CUMULATIVE_DOUBLE},
		{"gauge", textparse.MetricTypeGauge, metricspb.MetricDescriptor_GAUGE_DOUBLE},
		{"histogram", textparse.MetricTypeHistogram, metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION},
		{"guageHistogram", textparse.MetricTypeGaugeHistogram, metricspb.MetricDescriptor_GAUGE_DISTRIBUTION},
		{"summary", textparse.MetricTypeSummary, metricspb.MetricDescriptor_SUMMARY},
		{"info", textparse.MetricTypeInfo, metricspb.MetricDescriptor_UNSPECIFIED},
		{"stateset", textparse.MetricTypeStateset, metricspb.MetricDescriptor_UNSPECIFIED},
		{"unknown", textparse.MetricTypeUnknown, metricspb.MetricDescriptor_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convToOCAMetricType(tt.metricType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convToOCAMetricType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_metricBuilder(t *testing.T) {
	type pt struct {
		lb     labels.Labels
		v      float64
		hasErr bool
	}

	node := &commonpb.Node{
		ServiceInfo: &commonpb.ServiceInfo{Name: "myjob"},
		Identifier: &commonpb.ProcessIdentifier{
			HostName: "example.com",
		},
	}

	ts := int64(1555366610000)
	tsOc := timestampFromMs(ts)

	mc := &mockMetadataCache{
		data: map[string]scrape.MetricMetadata{
			"counter_test":    {Metric: "counter_test", Type: textparse.MetricTypeCounter, Help: "", Unit: ""},
			"gauge_test":      {Metric: "gauge_test", Type: textparse.MetricTypeGauge, Help: "", Unit: ""},
			"hist_test":       {Metric: "hist_test", Type: textparse.MetricTypeHistogram, Help: "", Unit: ""},
			"ghist_test":      {Metric: "ghist_test", Type: textparse.MetricTypeGaugeHistogram, Help: "", Unit: ""},
			"summary_test":    {Metric: "summary_test", Type: textparse.MetricTypeSummary, Help: "", Unit: ""},
			"unknown_test":    {Metric: "unknown_test", Type: textparse.MetricTypeUnknown, Help: "", Unit: ""},
			"poor_name_count": {Metric: "poor_name_count", Type: textparse.MetricTypeCounter, Help: "", Unit: ""},
			"up":              {Metric: "up", Type: textparse.MetricTypeCounter, Help: "", Unit: ""},
			"scrape_foo":      {Metric: "scrape_foo", Type: textparse.MetricTypeCounter, Help: "", Unit: ""},
		},
	}

	tests := []struct {
		name       string
		pts        []*pt
		processErr bool
		buildErr   bool
		metrics    []*metricspb.Metric
	}{
		{
			name: "counters",
			pts: []*pt{
				{createLabels("counter_test", "t1", "1"), 1.0, false},
				{createLabels("counter_test", "t2", "2"), 2.0, false},
				{createLabels("counter_test", "t1", "3", "t2", "4"), 3.0, false},
				{createLabels("counter_test"), 4.0, false},
			},
			buildErr: false,
			metrics: []*metricspb.Metric{
				{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name:      "counter_test",
						Type:      metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
						LabelKeys: []*metricspb.LabelKey{{Key: "t1"}, {Key: "t2"}}},
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "1", HasValue: true}, {Value: "", HasValue: false}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 1.0}},
							},
						},
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "", HasValue: false}, {Value: "2", HasValue: true}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 2.0}},
							},
						},
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "3", HasValue: true}, {Value: "4", HasValue: true}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 3.0}},
							},
						},
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "", HasValue: false}, {Value: "", HasValue: false}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 4.0}},
							},
						},
					},
				},
			},
		},
		{
			name: "gauge",
			pts: []*pt{
				{createLabels("gauge_test", "t1", "1"), 1.0, false},
				{createLabels("gauge_test", "t2", "2"), 2.0, false},
				{createLabels("gauge_test", "t1", "3", "t2", "4"), 3.0, false},
				{createLabels("gauge_test"), 4.0, false},
			},
			buildErr: false,
			metrics: []*metricspb.Metric{
				{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name:      "gauge_test",
						Type:      metricspb.MetricDescriptor_GAUGE_DOUBLE,
						LabelKeys: []*metricspb.LabelKey{{Key: "t1"}, {Key: "t2"}}},
					Timeseries: []*metricspb.TimeSeries{
						{
							LabelValues: []*metricspb.LabelValue{{Value: "1", HasValue: true}, {Value: "", HasValue: false}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 1.0}},
							},
						},
						{
							LabelValues: []*metricspb.LabelValue{{Value: "", HasValue: false}, {Value: "2", HasValue: true}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 2.0}},
							},
						},
						{
							LabelValues: []*metricspb.LabelValue{{Value: "3", HasValue: true}, {Value: "4", HasValue: true}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 3.0}},
							},
						},
						{
							LabelValues: []*metricspb.LabelValue{{Value: "", HasValue: false}, {Value: "", HasValue: false}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 4.0}},
							},
						},
					},
				},
			},
		},
		{
			name: "two_groups",
			pts: []*pt{
				{createLabels("counter_test", "t1", "1"), 1.0, false},
				{createLabels("counter_test", "t2", "2"), 2.0, false},
				{createLabels("gauge_test", "t1", "1"), 1.0, false},
				{createLabels("gauge_test", "t2", "2"), 2.0, false},
			},
			buildErr: false,
			metrics: []*metricspb.Metric{
				{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name:      "counter_test",
						Type:      metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
						LabelKeys: []*metricspb.LabelKey{{Key: "t1"}, {Key: "t2"}}},
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "1", HasValue: true}, {Value: "", HasValue: false}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 1.0}},
							},
						},
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "", HasValue: false}, {Value: "2", HasValue: true}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 2.0}},
							},
						},
					},
				},
				{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name:      "gauge_test",
						Type:      metricspb.MetricDescriptor_GAUGE_DOUBLE,
						LabelKeys: []*metricspb.LabelKey{{Key: "t1"}, {Key: "t2"}}},
					Timeseries: []*metricspb.TimeSeries{
						{
							LabelValues: []*metricspb.LabelValue{{Value: "1", HasValue: true}, {Value: "", HasValue: false}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 1.0}},
							},
						},
						{
							LabelValues: []*metricspb.LabelValue{{Value: "", HasValue: false}, {Value: "2", HasValue: true}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 2.0}},
							},
						},
					},
				},
			},
		},
		{
			name: "histogram",
			pts: []*pt{
				{createLabels("hist_test", "t1", "1", "le", "10"), 1.0, false},
				{createLabels("hist_test", "t1", "1", "le", "20"), 3.0, false},
				{createLabels("hist_test", "t1", "1", "le", "+inf"), 10.0, false},
				{createLabels("hist_test_sum", "t1", "1"), 100.0, false},
				{createLabels("hist_test_count", "t1", "1"), 10.0, false},
				{createLabels("hist_test", "t1", "2", "le", "10"), 10.0, false},
				{createLabels("hist_test", "t1", "2", "le", "20"), 30.0, false},
				{createLabels("hist_test", "t1", "2", "le", "+inf"), 100.0, false},
				{createLabels("hist_test_sum", "t1", "2"), 10000.0, false},
				{createLabels("hist_test_count", "t1", "2"), 100.0, false},
			},
			buildErr: false,
			metrics: []*metricspb.Metric{
				{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name:      "hist_test",
						Type:      metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
						LabelKeys: []*metricspb.LabelKey{{Key: "t1"}}},
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "1", HasValue: true}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DistributionValue{
									DistributionValue: &metricspb.DistributionValue{
										BucketOptions: &metricspb.DistributionValue_BucketOptions{
											Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
												Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
													Bounds: []float64{10, 20},
												},
											},
										},
										Count:   10,
										Sum:     100.0,
										Buckets: []*metricspb.DistributionValue_Bucket{{Count: 1}, {Count: 2}, {Count: 7}},
									}}},
							},
						},
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "2", HasValue: true}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DistributionValue{
									DistributionValue: &metricspb.DistributionValue{
										BucketOptions: &metricspb.DistributionValue_BucketOptions{
											Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
												Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
													Bounds: []float64{10, 20},
												},
											},
										},
										Count:   100,
										Sum:     10000.0,
										Buckets: []*metricspb.DistributionValue_Bucket{{Count: 10}, {Count: 20}, {Count: 70}},
									}}},
							},
						},
					},
				},
			},
		},
		{
			name: "gaugehistogram",
			pts: []*pt{
				{createLabels("ghist_test", "t1", "1", "le", "10"), 1.0, false},
				{createLabels("ghist_test", "t1", "1", "le", "20"), 3.0, false},
				{createLabels("ghist_test", "t1", "1", "le", "+inf"), 10.0, false},
				{createLabels("ghist_test_sum", "t1", "1"), 100.0, false},
				{createLabels("ghist_test_count", "t1", "1"), 10.0, false},
			},
			buildErr: false,
			metrics: []*metricspb.Metric{
				{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name:      "ghist_test",
						Type:      metricspb.MetricDescriptor_GAUGE_DISTRIBUTION,
						LabelKeys: []*metricspb.LabelKey{{Key: "t1"}}},
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "1", HasValue: true}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DistributionValue{
									DistributionValue: &metricspb.DistributionValue{
										BucketOptions: &metricspb.DistributionValue_BucketOptions{
											Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
												Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
													Bounds: []float64{10, 20},
												},
											},
										},
										Count:   10,
										Sum:     100.0,
										Buckets: []*metricspb.DistributionValue_Bucket{{Count: 1}, {Count: 2}, {Count: 7}},
									}}},
							},
						},
					},
				},
			},
		},
		{
			name: "histogram_mixed_oder",
			pts: []*pt{
				{createLabels("ghist_test", "t1", "1", "le", "10"), 1.0, false},
				{createLabels("ghist_test_sum", "t1", "1"), 100.0, false},
				{createLabels("ghist_test", "t1", "1", "le", "+inf"), 10.0, false},
				{createLabels("ghist_test_count", "t1", "1"), 10.0, false},
				{createLabels("ghist_test", "t1", "1", "le", "20"), 3.0, false},
			},
			buildErr: false,
			metrics: []*metricspb.Metric{
				{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name:      "ghist_test",
						Type:      metricspb.MetricDescriptor_GAUGE_DISTRIBUTION,
						LabelKeys: []*metricspb.LabelKey{{Key: "t1"}}},
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "1", HasValue: true}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DistributionValue{
									DistributionValue: &metricspb.DistributionValue{
										BucketOptions: &metricspb.DistributionValue_BucketOptions{
											Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
												Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
													Bounds: []float64{10, 20},
												},
											},
										},
										Count:   10,
										Sum:     100.0,
										Buckets: []*metricspb.DistributionValue_Bucket{{Count: 1}, {Count: 2}, {Count: 7}},
									}}},
							},
						},
					},
				},
			},
		},
		{
			name: "summary",
			pts: []*pt{
				{createLabels("summary_test", "t1", "1", "quantile", "0.5"), 1.0, false},
				{createLabels("summary_test", "t1", "1", "quantile", "0.9"), 3.0, false},
				{createLabels("summary_test_sum", "t1", "1"), 100.0, false},
				{createLabels("summary_test_count", "t1", "1"), 1000.0, false},
			},
			buildErr: false,
			metrics: []*metricspb.Metric{
				{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name:      "summary_test",
						Type:      metricspb.MetricDescriptor_SUMMARY,
						LabelKeys: []*metricspb.LabelKey{{Key: "t1"}}},
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "1", HasValue: true}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_SummaryValue{
									SummaryValue: &metricspb.SummaryValue{
										Sum:   &wrappers.DoubleValue{Value: 100.0},
										Count: &wrappers.Int64Value{Value: 1000},
										Snapshot: &metricspb.SummaryValue_Snapshot{
											PercentileValues: []*metricspb.SummaryValue_Snapshot_ValueAtPercentile{
												{Percentile: 50.0, Value: 1},
												{Percentile: 90.0, Value: 3},
											},
										}}}},
							},
						},
					},
				},
			},
		},
		{
			name: "unknowns",
			pts: []*pt{
				{createLabels("unknown_test", "t1", "1"), 1.0, false},
				{createLabels("unknown_test", "t2", "2"), 2.0, false},
				{createLabels("unknown_test", "t1", "3", "t2", "4"), 3.0, false},
				{createLabels("unknown_test"), 4.0, false},
			},
			buildErr: false,
			metrics: []*metricspb.Metric{
				{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name:      "unknown_test",
						Type:      metricspb.MetricDescriptor_UNSPECIFIED,
						LabelKeys: []*metricspb.LabelKey{{Key: "t1"}, {Key: "t2"}}},
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "1", HasValue: true}, {Value: "", HasValue: false}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 1.0}},
							},
						},
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "", HasValue: false}, {Value: "2", HasValue: true}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 2.0}},
							},
						},
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "3", HasValue: true}, {Value: "4", HasValue: true}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 3.0}},
							},
						},
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "", HasValue: false}, {Value: "", HasValue: false}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 4.0}},
							},
						},
					},
				},
			},
		},
		{
			name: "no-hints-individual-family",
			pts: []*pt{
				{createLabels("metric_family1", "t1", "1"), 1.0, false},
				{createLabels("metric_family2", "t2", "2"), 2.0, false},
			},
			buildErr: false,
			metrics: []*metricspb.Metric{
				{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name:      "metric_family1",
						Type:      metricspb.MetricDescriptor_UNSPECIFIED,
						LabelKeys: []*metricspb.LabelKey{{Key: "t1"}}},
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "1", HasValue: true}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 1.0}},
							},
						},
					},
				},
				{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name:      "metric_family2",
						Type:      metricspb.MetricDescriptor_UNSPECIFIED,
						LabelKeys: []*metricspb.LabelKey{{Key: "t2"}}},
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "2", HasValue: true}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 2.0}},
							},
						},
					},
				},
			},
		},
		{
			name: "poor_name_count",
			pts: []*pt{
				{createLabels("poor_name_count", "t1", "1"), 1.0, false},
				{createLabels("poor_name_count", "t2", "2"), 2.0, false},
			},
			buildErr: false,
			metrics: []*metricspb.Metric{
				{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name:      "poor_name_count",
						Type:      metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
						LabelKeys: []*metricspb.LabelKey{{Key: "t1"}, {Key: "t2"}}},
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "1", HasValue: true}, {Value: "", HasValue: false}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 1.0}},
							},
						},
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "", HasValue: false}, {Value: "2", HasValue: true}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 2.0}},
							},
						},
					},
				},
			},
		},
		{
			name: "poor_name_nohint_count",
			pts: []*pt{
				{createLabels("poor_name_nohint_count", "t1", "1"), 1.0, false},
				{createLabels("poor_name_nohint_count", "t2", "2"), 2.0, false},
			},
			buildErr: false,
			metrics: []*metricspb.Metric{
				{
					MetricDescriptor: &metricspb.MetricDescriptor{
						Name:      "poor_name_nohint_count",
						Type:      metricspb.MetricDescriptor_UNSPECIFIED,
						LabelKeys: []*metricspb.LabelKey{{Key: "t1"}, {Key: "t2"}}},
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "1", HasValue: true}, {Value: "", HasValue: false}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 1.0}},
							},
						},
						{
							StartTimestamp: tsOc,
							LabelValues:    []*metricspb.LabelValue{{Value: "", HasValue: false}, {Value: "2", HasValue: true}},
							Points: []*metricspb.Point{
								{Timestamp: tsOc, Value: &metricspb.Point_DoubleValue{DoubleValue: 2.0}},
							},
						},
					},
				},
			},
		},
		{
			name:     "noDataAdded",
			pts:      []*pt{},
			buildErr: true,
		},
		{
			name: "emptyMetricName",
			pts: []*pt{
				{createLabels("counter_test", "t1", "1"), 1.0, false},
				{createLabels("counter_test", "t2", "2"), 2.0, false},
				{createLabels("", "t1", "3", "t2", "4"), 3.0, true},
			},
		},
		{
			name: "bad_label_histogram",
			pts: []*pt{
				{createLabels("ghist_test", "t1", "1"), 1.0, true},
			},
			buildErr: true,
		},
		{
			name: "incomplete_histogram_no_sum_count",
			pts: []*pt{
				{createLabels("ghist_test", "t1", "1", "le", "10"), 1.0, false},
			},
			buildErr: true,
		},
		{
			name: "incomplete_histogram_no_buckets",
			pts: []*pt{
				{createLabels("ghist_test_sum", "t1", "1"), 1.0, false},
				{createLabels("ghist_test_count", "t1", "1"), 1.0, false},
			},
			buildErr: true,
		},
		{
			name: "bad_label_summary",
			pts: []*pt{
				{createLabels("summary_test", "t1", "1"), 1.0, true},
			},
			buildErr: true,
		},
		{
			name: "incomplete_summary_no_sum_count",
			pts: []*pt{
				{createLabels("summary_test", "t1", "1", "quantile", "0.5"), 1.0, false},
			},
			buildErr: true,
		},
		{
			name: "incomplete_summary_no_sum_count",
			pts: []*pt{
				{createLabels("summary_test_sum", "t1", "1"), 1.0, false},
				{createLabels("summary_test_count", "t1", "1"), 1.0, false},
			},
			buildErr: true,
		},
		{
			name: "incomplete_previous",
			pts: []*pt{
				{createLabels("summary_test", "t1", "1", "quantile", "0.5"), 1.0, false},
				{createLabels("new_metric", "t1", "1"), 1.0, true},
			},
		},
		{
			name: "skipped",
			pts: []*pt{
				{createLabels("up", "t1", "1"), 1.0, false},
				{createLabels("scrape_foo", "t1", "1"), 1.0, false},
			},
			buildErr: false,
			metrics:  make([]*metricspb.Metric, 0),
		},
		{
			name:     "nodata",
			pts:      []*pt{},
			buildErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newMetricBuilder(node, mc, testLogger)
			for _, v := range tt.pts {
				if err := b.AddDataPoint(v.lb, ts, v.v); (err != nil) != v.hasErr {
					t.Errorf("metricBuilder.AddDataPoint() error = %v, wantErr %v", err, v.hasErr)
				}
				if v.hasErr {
					// any error in between will cause whole page to fail on scrapeLoop
					// no need to continue
					return
				}
			}

			mt, err := b.Build()

			if err != nil {
				if !tt.buildErr {
					t.Errorf("metricBuilder.Build() error = %v, wantErr %v", err, tt.buildErr)
				}
				return
			} else if tt.buildErr {
				t.Errorf("metricBuilder.Build() error = %v, wantErr %v", err, tt.buildErr)
				return
			}

			if !reflect.DeepEqual(mt.Metrics, tt.metrics) {
				t.Errorf("metricBuilder.Build() metric = %v, want %v",
					string(exportertest.ToJSON(mt.Metrics)), string(exportertest.ToJSON(tt.metrics)))
			}
		})
	}
}

func createLabels(mFamily string, tagPairs ...string) labels.Labels {
	lm := make(map[string]string)
	lm[model.MetricNameLabel] = mFamily
	if len(tagPairs)%2 != 0 {
		panic("tag pairs is not even")
	}

	for i := 0; i < len(tagPairs); i += 2 {
		lm[tagPairs[i]] = tagPairs[i+1]
	}

	return labels.FromMap(lm)
}

func Test_heuristicalMetricAndKnownUnits(t *testing.T) {
	tests := []struct {
		metricName string
		parsedUnit string
		want       string
	}{
		{"test", "ms", "ms"},
		{"millisecond", "", ""},
		{"test_millisecond", "", "ms"},
		{"test_milliseconds", "", "ms"},
		{"test_ms", "", "ms"},
		{"test_second", "", "s"},
		{"test_seconds", "", "s"},
		{"test_s", "", "s"},
		{"test_microsecond", "", "us"},
		{"test_microseconds", "", "us"},
		{"test_us", "", "us"},
		{"test_nanosecond", "", "ns"},
		{"test_nanoseconds", "", "ns"},
		{"test_ns", "", "ns"},
		{"test_byte", "", "By"},
		{"test_bytes", "", "By"},
		{"test_by", "", "By"},
		{"test_bit", "", "Bi"},
		{"test_bits", "", "Bi"},
		{"test_kilogram", "", "kg"},
		{"test_kilograms", "", "kg"},
		{"test_kg", "", "kg"},
		{"test_gram", "", "g"},
		{"test_grams", "", "g"},
		{"test_g", "", "g"},
		{"test_nanogram", "", "ng"},
		{"test_nanograms", "", "ng"},
		{"test_ng", "", "ng"},
		{"test_meter", "", "m"},
		{"test_meters", "", "m"},
		{"test_metre", "", "m"},
		{"test_metres", "", "m"},
		{"test_m", "", "m"},
		{"test_kilometer", "", "km"},
		{"test_kilometers", "", "km"},
		{"test_kilometre", "", "km"},
		{"test_kilometres", "", "km"},
		{"test_km", "", "km"},
		{"test_milimeter", "", "mm"},
		{"test_milimeters", "", "mm"},
		{"test_milimetre", "", "mm"},
		{"test_milimetres", "", "mm"},
		{"test_mm", "", "mm"},
	}
	for _, tt := range tests {
		t.Run(tt.metricName, func(t *testing.T) {
			if got := heuristicalMetricAndKnownUnits(tt.metricName, tt.parsedUnit); got != tt.want {
				t.Errorf("heuristicalMetricAndKnownUnits() = %v, want %v", got, tt.want)
			}
		})
	}
}
