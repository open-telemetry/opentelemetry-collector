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

package pmetricjson

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"

	otlpcollectormetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/metrics/v1"
	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
)

func TestReadMetricsDataUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	value := &otlpmetrics.MetricsData{}
	assert.NoError(t, UnmarshalMetricsData([]byte(jsonStr), value))
	assert.EqualValues(t, &otlpmetrics.MetricsData{}, value)
}

func TestReadExportMetricsServiceRequestUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	value := &otlpcollectormetrics.ExportMetricsServiceRequest{}
	assert.NoError(t, UnmarshalExportMetricsServiceRequest([]byte(jsonStr), value))
	assert.EqualValues(t, &otlpcollectormetrics.ExportMetricsServiceRequest{}, value)
}

func TestExemplarIntVal(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    otlpmetrics.Exemplar
	}{
		{
			name:    "int",
			jsonStr: `{"as_int":1}`,
			want: otlpmetrics.Exemplar{
				Value: &otlpmetrics.Exemplar_AsInt{
					AsInt: 1,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.jsonStr))
			defer jsoniter.ConfigFastest.ReturnIterator(iter)
			got := readExemplar(iter)
			assert.EqualValues(t, tt.want, got)
		})
	}
}

func TestExemplarInvalidTraceID(t *testing.T) {
	jsonStr := `{"traceId":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readExemplar(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse trace_id")
	}
}

func TestExemplarInvalidSpanID(t *testing.T) {
	jsonStr := `{"spanId":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readExemplar(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse span_id")
	}
}

func TestExemplarUnknownField(t *testing.T) {
	jsonStr := `{"exists":"true"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	value := readExemplar(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, otlpmetrics.Exemplar{}, value)
}

func TestReadResourceMetricsUnknownField(t *testing.T) {
	jsonStr := `{"exists":"true"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	value := readResourceMetrics(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, &otlpmetrics.ResourceMetrics{}, value)
}

func TestReadInstrumentationLibraryMetricsUnknownField(t *testing.T) {
	jsonStr := `{"exists":"true"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	value := readScopeMetrics(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, &otlpmetrics.ScopeMetrics{}, value)
}

func TestReadInstrumentationLibraryUnknownField(t *testing.T) {
	jsonStr := `{"instrumentationLibrary":{"exists":"true"}}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	value := readScopeMetrics(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, &otlpmetrics.ScopeMetrics{}, value)
}

func TestReadMetricUnknownField(t *testing.T) {
	type args struct {
		jsonStr string
		want    *otlpmetrics.Metric
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "sum has unknown field",
			args: args{
				jsonStr: `{"sum":{"exists":"true"}}`,
				want: &otlpmetrics.Metric{
					Data: &otlpmetrics.Metric_Sum{
						Sum: &otlpmetrics.Sum{},
					},
				},
			},
		},
		{
			name: "gauge has unknown field",
			args: args{
				want: &otlpmetrics.Metric{
					Data: &otlpmetrics.Metric_Gauge{
						Gauge: &otlpmetrics.Gauge{},
					},
				},
				jsonStr: `{"gauge":{"exists":"true"}}`,
			},
		},
		{
			name: "histogram has unknown field",
			args: args{
				want: &otlpmetrics.Metric{
					Data: &otlpmetrics.Metric_Histogram{
						Histogram: &otlpmetrics.Histogram{},
					},
				},
				jsonStr: `{"histogram":{"exists":"true"}}`,
			},
		},
		{
			name: "exponential_histogram has unknown field",
			args: args{
				want: &otlpmetrics.Metric{
					Data: &otlpmetrics.Metric_ExponentialHistogram{
						ExponentialHistogram: &otlpmetrics.ExponentialHistogram{},
					},
				},
				jsonStr: `{"exponential_histogram":{"exists":"true"}}`,
			},
		},
		{
			name: "Summary has unknown field",
			args: args{
				want: &otlpmetrics.Metric{
					Data: &otlpmetrics.Metric_Summary{
						Summary: &otlpmetrics.Summary{},
					},
				},
				jsonStr: `{"summary":{"exists":"true"}}`,
			},
		},
		{
			name: "Metrics has unknown field",
			args: args{
				want:    &otlpmetrics.Metric{},
				jsonStr: `{"exists":{"exists":"true"}}`,
			},
		},
	}
	for _, tt := range tests {
		iter := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.args.jsonStr))
		jsoniter.ConfigFastest.ReturnIterator(iter)
		value := readMetric(iter)
		assert.NoError(t, iter.Error)
		assert.EqualValues(t, tt.args.want, value)
	}
}

func TestReadNumberDataPointUnknownField(t *testing.T) {
	jsonStr := `{"exists":{"exists":"true"}}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	value := readNumberDataPoint(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, &otlpmetrics.NumberDataPoint{}, value)
}

func TestReadHistogramDataPointUnknownField(t *testing.T) {
	jsonStr := `{"exists":{"exists":"true"},"count":3}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	value := readHistogramDataPoint(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, &otlpmetrics.HistogramDataPoint{
		Count: 3,
	}, value)
}

func TestReadExponentialHistogramDataPointUnknownField(t *testing.T) {
	jsonStr := `{"exists":{"exists":"true"},"count":3}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	value := readExponentialHistogramDataPoint(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, &otlpmetrics.ExponentialHistogramDataPoint{
		Count: 3,
	}, value)
}

func TestReadExponentialHistogramDataPointBucketsUnknownField(t *testing.T) {
	jsonStr := `{"exists":{"exists":"true"},"offset":3}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	value := readExponentialHistogramBuckets(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, otlpmetrics.ExponentialHistogramDataPoint_Buckets{
		Offset: 3,
	}, value)
}

func TestReadQuantileValue(t *testing.T) {
	jsonStr := `{"exists":{"exists":"true"},"value":3}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	value := readQuantileValue(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, &otlpmetrics.SummaryDataPoint_ValueAtQuantile{
		Value: 3,
	}, value)
}

func TestReadSummaryDataPoint(t *testing.T) {
	jsonStr := `{"exists":{"exists":"true"},"count":3}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	value := readSummaryDataPoint(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, &otlpmetrics.SummaryDataPoint{
		Count: 3,
	}, value)
}
