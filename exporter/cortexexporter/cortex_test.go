// Copyright 2020 The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cortexexporter

import (
	"github.com/prometheus/prometheus/prompb"
	common "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	"strconv"
	"testing"
	"reflect"

	"github.com/stretchr/testify/assert"

	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
	// "github.com/stretchr/testify/require"
)

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

func Test_addSample(t *testing.T) {
	type testCase struct{
		desc 	otlp.MetricDescriptor_Type
		sample prompb.Sample
		labels []prompb.Label
	}

	tests := []struct{
		name    	string
		orig   		map[string]*prompb.TimeSeries
		testCase 	[]testCase
		want   		map[string]*prompb.TimeSeries
	} {
		{
			name: "two_points_same_ts_same_metric",
			testCase: []testCase{
				{	otlp.MetricDescriptor_INT64,
					getSample(float64(int_val1), time1),
					 promlbs1.Labels,
				},
				{
					otlp.MetricDescriptor_INT64,
					getSample(float64(int_val2), time2),
					promlbs1.Labels,
				},
			},
			want: map[string]*prompb.TimeSeries {
					typeInt64+"-"+label11+"-"+value11+"-"+label21+"-"+value21:
							getTimeSeries(getPromLabels(label11,value11, label12,value12),
											getSample(float64(int_val1),time1),
											getSample(float64(int_val2),time2)),
			},
		},
		{
			name: "two_points_different_ts_same_metric",
			testCase: []testCase{
				{	otlp.MetricDescriptor_INT64,
					getSample(float64(int_val1), time1),
					promlbs1.Labels,
				},
				{	otlp.MetricDescriptor_INT64,
					getSample(float64(int_val1), time1),
					promlbs2.Labels,
				},
			},
			want: map[string]*prompb.TimeSeries {
				typeInt64+"-"+label11+"-"+value11+"-"+label21+"-"+value21:
				getTimeSeries(getPromLabels(label11,value11, label12,value12),
					getSample(float64(int_val1),time1),),
				typeInt64+"-"+label21+"-"+value21+"-"+label22+"-"+value22:
				getTimeSeries(getPromLabels(label21,value21, label22,value22),
					getSample(float64(int_val1),time2),),
			},
		},
	}

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addSample(tt.orig,&tt.testCase[0].sample, tt.testCase[0].labels, tt.testCase[0].desc)
			addSample(tt.orig,&tt.testCase[1].sample, tt.testCase[1].labels, tt.testCase[1].desc)
			assert.EqualValues(t, true, reflect.DeepEqual(tt.orig,tt.want))
		})
	}
}

func Test_timeSeriesSignature(t *testing.T) {
	tests := []struct {
		name	string
		lbs 	[]prompb.Label
		desc   	otlp.MetricDescriptor_Type
		want    string
	} {
		{
			"int64_signature",
			promlbs1.Labels,
			otlp.MetricDescriptor_INT64,
			typeInt64+"-"+label11+"-"+value11+"-"+label12+"-"+value12,
		},
		{

			"histogram_signature",
			promlbs2.Labels,
			otlp.MetricDescriptor_HISTOGRAM,
			typeHistogram+"-"+label21+"-"+value21+"-"+label22+"-"+value22,
		},
	}

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualValues(t, tt.want, timeSeriesSignature(tt.desc, &tt.lbs))
		})
	}
}

// Labels should be sanitized; label in extra overrides label in labels if collision happens
func Test_createLabelSet(t *testing.T) {
	tests := []struct {
		name string
		orig []*common.StringKeyValue
		extras []string
		want []prompb.Label
	} {
		{
			"labels_clean",
			lbs1,
			[]string{label31,value31,label32,value32},
			append(promlbs1.Labels,promlbs3.Labels...),
		},
		{
			"labels_duplicate_in_extras",
			lbs1,
			[]string{label11,value31},
			getPromLabels(label11,value31,label12, value12).Labels,
		},
		{
			"labels_dirty",
			lbs1Dirty,
			[]string{label31+dirty1,value31,label32,value32},
			getPromLabels(label11,value31,label12, value12).Labels,
		},
	}
	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualValues(t, true, reflect.DeepEqual(createLabelSet(tt.orig,tt.extras...),tt.want))
		})
	}
}
/*
func Test_handleScalarMetric(t *testing.T) {

	//tests := []handleMetricTest{}

	// two points in the same ts in the same metric

	// two points in different ts in the same metric

	// two points in the same ts in different metrics

	// two points in different ts in different metrics

}
*/