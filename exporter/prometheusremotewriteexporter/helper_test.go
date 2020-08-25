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

package prometheusremotewriteexporter

import (
	"strconv"
	"testing"

	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"

	common "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
)

// Test_validateMetrics checks validateMetrics return true if a type and temporality combination is valid, false
// otherwise.
func Test_validateMetrics(t *testing.T) {
	// define a single test
	type combTest struct {
		name string
		desc *otlp.MetricDescriptor
		want bool
	}

	tests := []combTest{}

	// append true cases
	for i := range validCombinations {
		name := "valid_" + strconv.Itoa(i)
		desc := getDescriptor(name, i, validCombinations)
		tests = append(tests, combTest{
			name,
			desc,
			true,
		})
	}
	// append false cases
	for i := range invalidCombinations {
		name := "invalid_" + strconv.Itoa(i)
		desc := getDescriptor(name, i, invalidCombinations)
		tests = append(tests, combTest{
			name,
			desc,
			false,
		})
	}
	// append nil case
	tests = append(tests, combTest{"invalid_nil", nil, false})

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateMetrics(tt.desc)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Test_addSample checks addSample updates the map it receives correctly based on the sample and Label
// set it receives.
// Test cases are two samples belonging to the same TimeSeries,  two samples belong to different TimeSeries, and nil
// case.
func Test_addSample(t *testing.T) {
	type testCase struct {
		desc   otlp.MetricDescriptor_Type
		sample prompb.Sample
		labels []prompb.Label
	}

	tests := []struct {
		name     string
		orig     map[string]*prompb.TimeSeries
		testCase []testCase
		want     map[string]*prompb.TimeSeries
	}{
		{
			"two_points_same_ts_same_metric",
			map[string]*prompb.TimeSeries{},
			[]testCase{
				{otlp.MetricDescriptor_INT64,
					getSample(float64(intVal1), msTime1),
					promLbs1,
				},
				{
					otlp.MetricDescriptor_INT64,
					getSample(float64(intVal2), msTime2),
					promLbs1,
				},
			},
			twoPointsSameTs,
		},
		{
			"two_points_different_ts_same_metric",
			map[string]*prompb.TimeSeries{},
			[]testCase{
				{otlp.MetricDescriptor_INT64,
					getSample(float64(intVal1), msTime1),
					promLbs1,
				},
				{otlp.MetricDescriptor_INT64,
					getSample(float64(intVal1), msTime2),
					promLbs2,
				},
			},
			twoPointsDifferentTs,
		},
	}
	t.Run("nil_case", func(t *testing.T) {
		tsMap := map[string]*prompb.TimeSeries{}
		addSample(tsMap, nil, nil, 0)
		assert.Exactly(t, tsMap, map[string]*prompb.TimeSeries{})
	})
	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addSample(tt.orig, &tt.testCase[0].sample, tt.testCase[0].labels, tt.testCase[0].desc)
			addSample(tt.orig, &tt.testCase[1].sample, tt.testCase[1].labels, tt.testCase[1].desc)
			assert.Exactly(t, tt.want, tt.orig)
		})
	}
}

// Test_timeSeries checks timeSeriesSignature returns consistent and unique signatures for a distinct label set and
// metric type combination.
func Test_timeSeriesSignature(t *testing.T) {
	tests := []struct {
		name string
		lbs  []prompb.Label
		desc otlp.MetricDescriptor_Type
		want string
	}{
		{
			"int64_signature",
			promLbs1,
			otlp.MetricDescriptor_INT64,
			typeInt64 + lb1Sig,
		},
		{
			"histogram_signature",
			promLbs2,
			otlp.MetricDescriptor_HISTOGRAM,
			typeHistogram + lb2Sig,
		},
		{
			"unordered_signature",
			getPromLabels(label22, value22, label21, value21),
			otlp.MetricDescriptor_HISTOGRAM,
			typeHistogram + lb2Sig,
		},
		// descriptor type cannot be nil, as checked by validateMetrics
		{
			"nil_case",
			nil,
			otlp.MetricDescriptor_HISTOGRAM,
			typeHistogram,
		},
	}

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualValues(t, tt.want, timeSeriesSignature(tt.desc, &tt.lbs))
		})
	}
}

// Test_createLabelSet checks resultant label names are sanitized and label in extra overrides label in labels if
// collision happens. It does not check whether labels are not sorted
func Test_createLabelSet(t *testing.T) {
	tests := []struct {
		name   string
		orig   []*common.StringKeyValue
		extras []string
		want   []prompb.Label
	}{
		{
			"labels_clean",
			lbs1,
			[]string{label31, value31, label32, value32},
			getPromLabels(label11, value11, label12, value12, label31, value31, label32, value32),
		},
		{
			"labels_duplicate_in_extras",
			lbs1,
			[]string{label11, value31},
			getPromLabels(label11, value31, label12, value12),
		},
		{
			"labels_dirty",
			lbs1Dirty,
			[]string{label31 + dirty1, value31, label32, value32},
			getPromLabels(label11+"_", value11, "key_"+label12, value12, label31+"_", value31, label32, value32),
		},
		{
			"no_original_case",
			nil,
			[]string{label31, value31, label32, value32},
			getPromLabels(label31, value31, label32, value32),
		},
		{
			"empty_extra_case",
			lbs1,
			[]string{"", ""},
			getPromLabels(label11, value11, label12, value12, "", ""),
		},
		{
			"single_left_over_case",
			lbs1,
			[]string{label31, value31, label32},
			getPromLabels(label11, value11, label12, value12, label31, value31),
		},
	}
	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ElementsMatch(t, tt.want, createLabelSet(tt.orig, tt.extras...))
		})
	}
}

// Tes_getPromMetricName checks if OTLP metric names are converted to Cortex metric names correctly.
// Test cases are empty namespace, monotonic metrics that require a total suffix, and metric names that contains
// invalid characters.
func Test_getPromMetricName(t *testing.T) {
	tests := []struct {
		name string
		desc *otlp.MetricDescriptor
		ns   string
		want string
	}{
		{
			"nil_case",
			nil,
			ns1,
			"",
		},
		{
			"normal_case",
			getDescriptor(name1, histogramComb, validCombinations),
			ns1,
			"test_ns_valid_single_int_point",
		},
		{
			"empty_namespace",
			getDescriptor(name1, summaryComb, validCombinations),
			"",
			"valid_single_int_point",
		},
		{
			"total_suffix",
			getDescriptor(name1, monotonicInt64Comb, validCombinations),
			ns1,
			"test_ns_valid_single_int_point_total",
		},
		{
			"dirty_string",
			getDescriptor(name1+dirty1, monotonicInt64Comb, validCombinations),
			"7" + ns1,
			"key_7test_ns_valid_single_int_point__total",
		},
	}
	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getPromMetricName(tt.desc, tt.ns))
		})
	}

}
