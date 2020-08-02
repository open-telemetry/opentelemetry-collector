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

	"github.com/stretchr/testify/assert"

	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
	// "github.com/stretchr/testify/require"
)
// TODO: Test_fix handleScalarPoint; right now metric name is not included in label
// TODO: make sure nil case is checked in every test

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
			 "two_points_same_ts_same_metric",
			 map[string]*prompb.TimeSeries {},
			[]testCase{
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
			map[string]*prompb.TimeSeries {
					typeInt64+"-"+label11+"-"+value11+"-"+label21+"-"+value21:
							getTimeSeries(getPromLabels(label11,value11, label12,value12),
											getSample(float64(int_val1),time1),
											getSample(float64(int_val2),time2)),
			},
		},
		{
			"two_points_different_ts_same_metric",
			map[string]*prompb.TimeSeries {},
			 []testCase{
				{	otlp.MetricDescriptor_INT64,
					getSample(float64(int_val1), time1),
					promlbs1.Labels,
				},
				{	otlp.MetricDescriptor_INT64,
					getSample(float64(int_val1), time1),
					promlbs2.Labels,
				},
			},
			twoPointsDifferentTs,
		},
	}

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addSample(tt.orig,&tt.testCase[0].sample, tt.testCase[0].labels, tt.testCase[0].desc)
			addSample(tt.orig,&tt.testCase[1].sample, tt.testCase[1].labels, tt.testCase[1].desc)
			assert.Exactly(t, tt.want, tt.orig)
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
			assert.Exactly(t, tt.want, createLabelSet(tt.orig,tt.extras...))
		})
	}
}

// Check if there is an error, the map stays the same
func Test_handleScalarMetric(t *testing.T) {
	tests := []struct {
		name 		string
		m			*otlp.Metric
		returnError  bool
		want   		map[string]*prompb.TimeSeries
	} {
		{
			"invalid_nil_array",
			&otlp.Metric{
				MetricDescriptor:    getDescriptor("valid_single_int_point",monotonicInt64,validCombinations),
				Int64DataPoints:     nil,
				DoubleDataPoints:    nil,
				HistogramDataPoints: nil,
				SummaryDataPoints:   nil,
			},
			true,
			map[string]*prompb.TimeSeries {},
		},
		{
			"same_ts_int_points",
			&otlp.Metric{
				MetricDescriptor:    getDescriptor("valid_single_int_point",monotonicInt64,validCombinations),
				Int64DataPoints:     []*otlp.Int64DataPoint{
					getIntDataPoint(lbs1,int_val1,time1),
					getIntDataPoint(lbs1,int_val2,time1),
				},
				DoubleDataPoints:    nil,
				HistogramDataPoints: nil,
				SummaryDataPoints:   nil,
			},
			false,
			twoPointsSameTs,
		},
		{
			"different_ts_int_points",
			&otlp.Metric{
				MetricDescriptor:    getDescriptor("valid_single_int_point",monotonicInt64,validCombinations),
				Int64DataPoints:     []*otlp.Int64DataPoint{
					getIntDataPoint(lbs1,int_val1,time1),
					getIntDataPoint(lbs2,int_val2,time2),
				},
				DoubleDataPoints:    nil,
				HistogramDataPoints: nil,
				SummaryDataPoints:   nil,
			},
			false,
			twoPointsDifferentTs,
		},
	}
	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tsMap := map[string]*prompb.TimeSeries{}
			ok := handleScalarMetric(tsMap, tt.m)
			if tt.returnError {
				assert.Nil(t,ok)
			}
			assert.Exactly(t, tt.want, tsMap)
		})
	}
}

func Test_handleHistogramMetric(t *testing.T) {
	sum := "sum"
	count := "count"
	bucket1 := "bucket1"
	bucket2 := "bucket2"
	bucketInf := "bucketInf"
	histPoint := otlp.HistogramDataPoint{
		Labels:            lbs1,
		StartTimeUnixNano: 0,
		TimeUnixNano:      uint64(time1.UnixNano()),
		Count:             uint64(int_val2),
		Sum:               float_val2,
		Buckets:           []*otlp.HistogramDataPoint_Bucket{
				{uint64(int_val1),
					nil,
				},
				{uint64(int_val1),
				nil,
				},
		},
		ExplicitBounds:    []float64{
			float_val1,
			float_val2,
		},
	}
	sigs := map[string]string{
		sum: typeHistogram+"-"+label11+"-"+value11+"-"+label21+"-"+value21+"-name-valid_single_point_sum",
		count: typeHistogram+"-"+label11+"-"+value11+"-"+label21+"-"+value21+"-name-valid_single_point_count",
		bucket1: typeHistogram+"-"+label11+"-"+value11+"-"+label21+"-"+value21+"-"+"le-"+
			strconv.FormatFloat(float_val1,'f',-1,64)+"-name-valid_single_point_bucket",
		bucket2: typeHistogram+"-"+label11+"-"+value11+"-"+label21+"-"+value21+"-"+"le-"+
			strconv.FormatFloat(float_val2,'f',-1,64)+"-name-valid_single_point_bucket",
		bucketInf: typeHistogram+"-"+label11+"-"+value11+"-"+label21+"-"+value21+"-"+"le-"+
			"+Inf"+"-name-valid_single_point_bucket",
	}
	lbls := map[string][]prompb.Label {
		sum: append(promlbs1.Labels,getPromLabels("name", "valid_single_point_sum").Labels...),
		count: append(promlbs1.Labels,getPromLabels("name", "valid_single_point_count").Labels...),
		bucket1:append(promlbs1.Labels,getPromLabels("name", "valid_single_point_bucket","le",
					strconv.FormatFloat(float_val1,'f',-1,64)).Labels...),
		bucket2: append(promlbs1.Labels,getPromLabels("name", "valid_single_point_bucket","le",
			strconv.FormatFloat(float_val2,'f',-1,64)).Labels...),
		bucketInf: append(promlbs1.Labels,getPromLabels("name", "valid_single_point_bucket","le",
			"+Inf").Labels...),
	}
	tests := []struct {
		name 		string
		m    		otlp.Metric
		returnError	bool
		want 		map[string]*prompb.TimeSeries
	} {
		{
			"invalid_nil_array",
			otlp.Metric{
				MetricDescriptor:    getDescriptor("invalid_nil_array", histogram, validCombinations),
				Int64DataPoints:     nil,
				DoubleDataPoints:    nil,
				HistogramDataPoints: nil,
				SummaryDataPoints:   nil,
			},
			true,
			map[string]*prompb.TimeSeries{},
		},
		{
			"single_histogram_point",
			otlp.Metric{
				MetricDescriptor:    getDescriptor("valid_single_int_point", histogram, validCombinations),
				Int64DataPoints:     nil,
				DoubleDataPoints:    nil,
				HistogramDataPoints: []*otlp.HistogramDataPoint{&histPoint},
				SummaryDataPoints:   nil,
			},
			false,
			map[string]*prompb.TimeSeries{
				
					sigs[sum]: &prompb.TimeSeries{
						Labels:      		  lbls[sum],
						Samples:              []prompb.Sample{getSample(float_val2,time1)},
						XXX_NoUnkeyedLiteral: struct{}{},
						XXX_unrecognized:     nil,
						XXX_sizecache:        0,
					},
					sigs[count]: &prompb.TimeSeries{
						Labels:      		  lbls[count],
						Samples:              []prompb.Sample{getSample(float64(int_val2),time1)},
						XXX_NoUnkeyedLiteral: struct{}{},
						XXX_unrecognized:     nil,
						XXX_sizecache:        0,
					},
					sigs[bucket1]: &prompb.TimeSeries{
						Labels:      		  lbls[bucket1],
						Samples:              []prompb.Sample{getSample(float64(int_val1),time1)},
						XXX_NoUnkeyedLiteral: struct{}{},
						XXX_unrecognized:     nil,
						XXX_sizecache:        0,
					},
					sigs[bucket2]: &prompb.TimeSeries{
						Labels:      		  lbls[bucket2],
						Samples:              []prompb.Sample{getSample(float64(int_val1),time1)},
						XXX_NoUnkeyedLiteral: struct{}{},
						XXX_unrecognized:     nil,
						XXX_sizecache:        0,
					},
					sigs[bucketInf]: &prompb.TimeSeries{
						Labels:      		  lbls[bucketInf],
						Samples:              []prompb.Sample{getSample(float64(int_val2),time1)},
						XXX_NoUnkeyedLiteral: struct{}{},
						XXX_unrecognized:     nil,
						XXX_sizecache:        0,
					},
			},
		},
	}
	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tsMap := map[string]*prompb.TimeSeries{}
			ok := handleScalarMetric(tsMap, &tt.m)
			if tt.returnError {
				assert.Nil(t,ok)
			}
			assert.Exactly(t, tt.want, tsMap)
		})
	}
}

