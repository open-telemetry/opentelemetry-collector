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
	"reflect"
	"runtime"
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/prometheus/prometheus/scrape"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const startTs = int64(1555366610000)
const interval = int64(15 * 1000)
const defaultBuilderStartTime = float64(1.0)

var testMetadata = map[string]scrape.MetricMetadata{
	"counter_test":    {Metric: "counter_test", Type: textparse.MetricTypeCounter, Help: "", Unit: ""},
	"counter_test2":   {Metric: "counter_test2", Type: textparse.MetricTypeCounter, Help: "", Unit: ""},
	"gauge_test":      {Metric: "gauge_test", Type: textparse.MetricTypeGauge, Help: "", Unit: ""},
	"gauge_test2":     {Metric: "gauge_test2", Type: textparse.MetricTypeGauge, Help: "", Unit: ""},
	"hist_test":       {Metric: "hist_test", Type: textparse.MetricTypeHistogram, Help: "", Unit: ""},
	"hist_test2":      {Metric: "hist_test2", Type: textparse.MetricTypeHistogram, Help: "", Unit: ""},
	"ghist_test":      {Metric: "ghist_test", Type: textparse.MetricTypeGaugeHistogram, Help: "", Unit: ""},
	"summary_test":    {Metric: "summary_test", Type: textparse.MetricTypeSummary, Help: "", Unit: ""},
	"summary_test2":   {Metric: "summary_test2", Type: textparse.MetricTypeSummary, Help: "", Unit: ""},
	"unknown_test":    {Metric: "unknown_test", Type: textparse.MetricTypeUnknown, Help: "", Unit: ""},
	"poor_name_count": {Metric: "poor_name_count", Type: textparse.MetricTypeCounter, Help: "", Unit: ""},
	"up":              {Metric: "up", Type: textparse.MetricTypeCounter, Help: "", Unit: ""},
	"scrape_foo":      {Metric: "scrape_foo", Type: textparse.MetricTypeCounter, Help: "", Unit: ""},
	"example_process_start_time_seconds": {Metric: "example_process_start_time_seconds",
		Type: textparse.MetricTypeGauge, Help: "", Unit: ""},
	"process_start_time_seconds": {Metric: "process_start_time_seconds",
		Type: textparse.MetricTypeGauge, Help: "", Unit: ""},
	"badprocess_start_time_seconds": {Metric: "badprocess_start_time_seconds",
		Type: textparse.MetricTypeGauge, Help: "", Unit: ""},
}

type testDataPoint struct {
	lb labels.Labels
	t  int64
	v  float64
}

type testScrapedPage struct {
	pts []*testDataPoint
}

type buildTestData struct {
	name   string
	inputs []*testScrapedPage
	wants  [][]*metricspb.Metric
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

func createDataPoint(mname string, value float64, tagPairs ...string) *testDataPoint {
	return &testDataPoint{
		lb: createLabels(mname, tagPairs...),
		v:  value,
	}
}

func runBuilderTests(t *testing.T, tests []buildTestData) {
	/*
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.EqualValues(t, len(tt.wants), len(tt.inputs))
				mc := newMockMetadataCache(testMetadata)
				st := startTs
				for i, page := range tt.inputs {
					b := newMetricBuilder(mc, true, "", testLogger, dummyStalenessStore())
					b.startTime = defaultBuilderStartTime // set to a non-zero value
					for _, pt := range page.pts {
						// set ts for testing
						pt.t = st
						assert.NoError(t, b.AddDataPoint(pt.lb, pt.t, pt.v))
					}
					metrics, _, _, err := b.Build()
					assert.NoError(t, err)
					assert.EqualValues(t, tt.wants[i], metrics)
					st += interval
				}
			})
		}
	*/
}

func runBuilderStartTimeTests(t *testing.T, tests []buildTestData,
	startTimeMetricRegex string, expectedBuilderStartTime float64) {
	/*
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mc := newMockMetadataCache(testMetadata)
				st := startTs
				for _, page := range tt.inputs {
					b := newMetricBuilder(mc, true, startTimeMetricRegex, testLogger, dummyStalenessStore())
					b.startTime = defaultBuilderStartTime // set to a non-zero value
					for _, pt := range page.pts {
						// set ts for testing
						pt.t = st
						assert.NoError(t, b.AddDataPoint(pt.lb, pt.t, pt.v))
					}
					_, _, _, err := b.Build()
					assert.NoError(t, err)
					assert.EqualValues(t, b.startTime, expectedBuilderStartTime)
					st += interval
				}
			})
		}
	*/
}

func Test_startTimeMetricMatch(t *testing.T) {
	matchBuilderStartTime := 123.456
	matchTests := []buildTestData{
		{
			name: "prefix_match",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("example_process_start_time_seconds",
							matchBuilderStartTime, "foo", "bar"),
					},
				},
			},
		},
		{
			name: "match",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("process_start_time_seconds",
							matchBuilderStartTime, "foo", "bar"),
					},
				},
			},
		},
	}
	nomatchTests := []buildTestData{
		{
			name: "nomatch1",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("_process_start_time_seconds",
							matchBuilderStartTime, "foo", "bar"),
					},
				},
			},
		},
		{
			name: "nomatch2",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("subprocess_start_time_seconds",
							matchBuilderStartTime, "foo", "bar"),
					},
				},
			},
		},
	}

	runBuilderStartTimeTests(t, matchTests, "^(.+_)*process_start_time_seconds$", matchBuilderStartTime)
	runBuilderStartTimeTests(t, nomatchTests, "^(.+_)*process_start_time_seconds$", defaultBuilderStartTime)
}

func Test_metricBuilder_summary(t *testing.T) {
	tests := []buildTestData{
		{
			name: "no-sum-and-count",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("summary_test", 5, "foo", "bar", "quantile", "1"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{},
			},
		},
		{
			name: "no-count",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("summary_test", 1, "foo", "bar", "quantile", "0.5"),
						createDataPoint("summary_test", 2, "foo", "bar", "quantile", "0.75"),
						createDataPoint("summary_test", 5, "foo", "bar", "quantile", "1"),
						createDataPoint("summary_test_sum", 500, "foo", "bar"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{},
			},
		},
		{
			name: "no-sum",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("summary_test", 1, "foo", "bar", "quantile", "0.5"),
						createDataPoint("summary_test", 2, "foo", "bar", "quantile", "0.75"),
						createDataPoint("summary_test", 5, "foo", "bar", "quantile", "1"),
						createDataPoint("summary_test_count", 500, "foo", "bar"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "summary_test",
							Type:      metricspb.MetricDescriptor_SUMMARY,
							LabelKeys: []*metricspb.LabelKey{{Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{{Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_SummaryValue{
										SummaryValue: &metricspb.SummaryValue{
											Sum:   &wrapperspb.DoubleValue{Value: 0.0},
											Count: &wrapperspb.Int64Value{Value: 500},
											Snapshot: &metricspb.SummaryValue_Snapshot{
												PercentileValues: []*metricspb.SummaryValue_Snapshot_ValueAtPercentile{
													{Percentile: 50.0, Value: 1},
													{Percentile: 75.0, Value: 2},
													{Percentile: 100.0, Value: 5},
												},
											}}}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "empty-quantiles",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("summary_test_sum", 100, "foo", "bar"),
						createDataPoint("summary_test_count", 500, "foo", "bar"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "summary_test",
							Type:      metricspb.MetricDescriptor_SUMMARY,
							LabelKeys: []*metricspb.LabelKey{{Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{{Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{
										Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_SummaryValue{
											SummaryValue: &metricspb.SummaryValue{
												Sum:   &wrapperspb.DoubleValue{Value: 100.0},
												Count: &wrapperspb.Int64Value{Value: 500},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "regular-summary",
			inputs: []*testScrapedPage{
				{
					pts: []*testDataPoint{
						createDataPoint("summary_test", 1, "foo", "bar", "quantile", "0.5"),
						createDataPoint("summary_test", 2, "foo", "bar", "quantile", "0.75"),
						createDataPoint("summary_test", 5, "foo", "bar", "quantile", "1"),
						createDataPoint("summary_test_sum", 100, "foo", "bar"),
						createDataPoint("summary_test_count", 500, "foo", "bar"),
					},
				},
			},
			wants: [][]*metricspb.Metric{
				{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name:      "summary_test",
							Type:      metricspb.MetricDescriptor_SUMMARY,
							LabelKeys: []*metricspb.LabelKey{{Key: "foo"}}},
						Timeseries: []*metricspb.TimeSeries{
							{
								StartTimestamp: timestampFromMs(startTs),
								LabelValues:    []*metricspb.LabelValue{{Value: "bar", HasValue: true}},
								Points: []*metricspb.Point{
									{Timestamp: timestampFromMs(startTs), Value: &metricspb.Point_SummaryValue{
										SummaryValue: &metricspb.SummaryValue{
											Sum:   &wrapperspb.DoubleValue{Value: 100.0},
											Count: &wrapperspb.Int64Value{Value: 500},
											Snapshot: &metricspb.SummaryValue_Snapshot{
												PercentileValues: []*metricspb.SummaryValue_Snapshot_ValueAtPercentile{
													{Percentile: 50.0, Value: 1},
													{Percentile: 75.0, Value: 2},
													{Percentile: 100.0, Value: 5},
												},
											}}}},
								},
							},
						},
					},
				},
			},
		},
	}

	runBuilderTests(t, tests)
}

func Test_metricBuilder_baddata(t *testing.T) {
	t.Run("empty-metric-name", func(t *testing.T) {
		mc := newMockMetadataCache(testMetadata)
		b := newMetricBuilderPdata(mc, true, "", testLogger, dummyStalenessStore())
		b.startTime = 1.0 // set to a non-zero value
		if err := b.AddDataPoint(labels.FromStrings("a", "b"), startTs, 123); err != errMetricNameNotFound {
			t.Error("expecting errMetricNameNotFound error, but get nil")
			return
		}

		if _, _, _, err := b.Build(); err != errNoDataToBuild {
			t.Error("expecting errNoDataToBuild error, but get nil")
		}
	})

	t.Run("histogram-datapoint-no-bucket-label", func(t *testing.T) {
		mc := newMockMetadataCache(testMetadata)
		b := newMetricBuilderPdata(mc, true, "", testLogger, dummyStalenessStore())
		b.startTime = 1.0 // set to a non-zero value
		if err := b.AddDataPoint(createLabels("hist_test", "k", "v"), startTs, 123); err != errEmptyBoundaryLabel {
			t.Error("expecting errEmptyBoundaryLabel error, but get nil")
		}
	})

	t.Run("summary-datapoint-no-quantile-label", func(t *testing.T) {
		mc := newMockMetadataCache(testMetadata)
		b := newMetricBuilderPdata(mc, true, "", testLogger, dummyStalenessStore())
		b.startTime = 1.0 // set to a non-zero value
		if err := b.AddDataPoint(createLabels("summary_test", "k", "v"), startTs, 123); err != errEmptyBoundaryLabel {
			t.Error("expecting errEmptyBoundaryLabel error, but get nil")
		}
	})

}

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

func Benchmark_dpgSignature(b *testing.B) {
	knownLabelKeys := []string{"a", "b"}
	labels := labels.FromStrings("a", "va", "b", "vb", "x", "xa")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		runtime.KeepAlive(dpgSignature(knownLabelKeys, labels))
	}
}

func Test_dpgSignature(t *testing.T) {
	knownLabelKeys := []string{"a", "b"}

	tests := []struct {
		name string
		ls   labels.Labels
		want string
	}{
		{"1st label", labels.FromStrings("a", "va"), `"a=va"`},
		{"2nd label", labels.FromStrings("b", "vb"), `"b=vb"`},
		{"two labels", labels.FromStrings("a", "va", "b", "vb"), `"a=va""b=vb"`},
		{"extra label", labels.FromStrings("a", "va", "b", "vb", "x", "xa"), `"a=va""b=vb"`},
		{"different order", labels.FromStrings("b", "vb", "a", "va"), `"a=va""b=vb"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dpgSignature(knownLabelKeys, tt.ls); got != tt.want {
				t.Errorf("dpgSignature() = %q, want %q", got, tt.want)
			}
		})
	}

	// this is important for caching start values, as new metrics with new tag of a same group can come up in a 2nd run,
	// however, its order within the group is not predictable. we need to have a way to generate a stable key even if
	// the total number of keys changes in between different scrape runs
	t.Run("knownLabelKeys updated", func(t *testing.T) {
		ls := labels.FromStrings("a", "va")
		want := dpgSignature(knownLabelKeys, ls)
		got := dpgSignature(append(knownLabelKeys, "c"), ls)
		if got != want {
			t.Errorf("dpgSignature() = %v, want %v", got, want)
		}
	})
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
		{"total", "foo_total", "foo"},
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
		{"guageHistogram", textparse.MetricTypeGaugeHistogram, metricspb.MetricDescriptor_UNSPECIFIED},
		{"summary", textparse.MetricTypeSummary, metricspb.MetricDescriptor_SUMMARY},
		{"info", textparse.MetricTypeInfo, metricspb.MetricDescriptor_UNSPECIFIED},
		{"stateset", textparse.MetricTypeStateset, metricspb.MetricDescriptor_UNSPECIFIED},
		{"unknown", textparse.MetricTypeUnknown, metricspb.MetricDescriptor_GAUGE_DOUBLE},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convToOCAMetricType(tt.metricType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convToOCAMetricType() = %v, want %v", got, tt.want)
			}
		})
	}
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

// Ensure that we reject duplicate label keys. See https://github.com/open-telemetry/wg-prometheus/issues/44.
func TestMetricBuilderDuplicateLabelKeysAreRejected(t *testing.T) {
	mc := newMockMetadataCache(testMetadata)
	mb := newMetricBuilderPdata(mc, true, "", testLogger, dummyStalenessStore())

	dupLabels := labels.Labels{
		{Name: "__name__", Value: "test"},
		{Name: "a", Value: "1"},
		{Name: "a", Value: "1"},
		{Name: "z", Value: "9"},
		{Name: "z", Value: "1"},
		{Name: "instance", Value: "0.0.0.0:8855"},
		{Name: "job", Value: "test"},
	}

	err := mb.AddDataPoint(dupLabels, 1917, 1.0)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), `invalid sample: non-unique label names: ["a" "z"]`)
}
