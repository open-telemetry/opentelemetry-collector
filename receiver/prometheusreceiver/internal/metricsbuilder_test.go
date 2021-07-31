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
	"github.com/stretchr/testify/assert"
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

func runBuilderStartTimeTests(t *testing.T, tests []buildTestDataPdata,
	startTimeMetricRegex string, expectedBuilderStartTime float64) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := newMockMetadataCache(testMetadata)
			st := startTs
			for _, page := range tt.inputs {
				b := newMetricBuilderPdata(mc, true, startTimeMetricRegex, testLogger, dummyStalenessStore())
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
}

func Test_startTimeMetricMatch(t *testing.T) {
	matchBuilderStartTime := 123.456
	matchTests := []buildTestDataPdata{
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
	nomatchTests := []buildTestDataPdata{
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
