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

	"github.com/stretchr/testify/assert"

	otlpcollectormetrics "go.opentelemetry.io/collector/model/internal/data/protogen/collector/metrics/v1"
	otlpcollectortrace "go.opentelemetry.io/collector/model/internal/data/protogen/collector/trace/v1"
	otlpcommon "go.opentelemetry.io/collector/model/internal/data/protogen/common/v1"
	otlpmetrics "go.opentelemetry.io/collector/model/internal/data/protogen/metrics/v1"
	otlptrace "go.opentelemetry.io/collector/model/internal/data/protogen/trace/v1"
)

func TestDeprecatedStatusCode(t *testing.T) {
	// See specification for handling status code here:
	// https://github.com/open-telemetry/opentelemetry-proto/blob/59c488bfb8fb6d0458ad6425758b70259ff4a2bd/opentelemetry/proto/trace/v1/trace.proto#L231
	tests := []struct {
		sendCode               otlptrace.Status_StatusCode
		sendDeprecatedCode     otlptrace.Status_DeprecatedStatusCode
		expectedRcvCode        otlptrace.Status_StatusCode
		expectedDeprecatedCode otlptrace.Status_DeprecatedStatusCode
	}{
		{
			// If code==STATUS_CODE_UNSET then the value of `deprecated_code` is the
			//   carrier of the overall status according to these rules:
			//
			//     if deprecated_code==DEPRECATED_STATUS_CODE_OK then the receiver MUST interpret
			//     the overall status to be STATUS_CODE_UNSET.
			sendCode:               otlptrace.Status_STATUS_CODE_UNSET,
			sendDeprecatedCode:     otlptrace.Status_DEPRECATED_STATUS_CODE_OK,
			expectedRcvCode:        otlptrace.Status_STATUS_CODE_UNSET,
			expectedDeprecatedCode: otlptrace.Status_DEPRECATED_STATUS_CODE_OK,
		},
		{
			//     if deprecated_code!=DEPRECATED_STATUS_CODE_OK then the receiver MUST interpret
			//     the overall status to be STATUS_CODE_ERROR.
			sendCode:               otlptrace.Status_STATUS_CODE_UNSET,
			sendDeprecatedCode:     otlptrace.Status_DEPRECATED_STATUS_CODE_ABORTED,
			expectedRcvCode:        otlptrace.Status_STATUS_CODE_ERROR,
			expectedDeprecatedCode: otlptrace.Status_DEPRECATED_STATUS_CODE_ABORTED,
		},
		{
			//   If code!=STATUS_CODE_UNSET then the value of `deprecated_code` MUST be
			//   overwritten, the `code` field is the sole carrier of the status.
			sendCode:               otlptrace.Status_STATUS_CODE_OK,
			sendDeprecatedCode:     otlptrace.Status_DEPRECATED_STATUS_CODE_OK,
			expectedRcvCode:        otlptrace.Status_STATUS_CODE_OK,
			expectedDeprecatedCode: otlptrace.Status_DEPRECATED_STATUS_CODE_OK,
		},
		{
			//   If code!=STATUS_CODE_UNSET then the value of `deprecated_code` MUST be
			//   overwritten, the `code` field is the sole carrier of the status.
			sendCode:               otlptrace.Status_STATUS_CODE_OK,
			sendDeprecatedCode:     otlptrace.Status_DEPRECATED_STATUS_CODE_UNKNOWN_ERROR,
			expectedRcvCode:        otlptrace.Status_STATUS_CODE_OK,
			expectedDeprecatedCode: otlptrace.Status_DEPRECATED_STATUS_CODE_OK,
		},
		{
			//   If code!=STATUS_CODE_UNSET then the value of `deprecated_code` MUST be
			//   overwritten, the `code` field is the sole carrier of the status.
			sendCode:               otlptrace.Status_STATUS_CODE_ERROR,
			sendDeprecatedCode:     otlptrace.Status_DEPRECATED_STATUS_CODE_OK,
			expectedRcvCode:        otlptrace.Status_STATUS_CODE_ERROR,
			expectedDeprecatedCode: otlptrace.Status_DEPRECATED_STATUS_CODE_UNKNOWN_ERROR,
		},
		{
			//   If code!=STATUS_CODE_UNSET then the value of `deprecated_code` MUST be
			//   overwritten, the `code` field is the sole carrier of the status.
			sendCode:               otlptrace.Status_STATUS_CODE_ERROR,
			sendDeprecatedCode:     otlptrace.Status_DEPRECATED_STATUS_CODE_UNKNOWN_ERROR,
			expectedRcvCode:        otlptrace.Status_STATUS_CODE_ERROR,
			expectedDeprecatedCode: otlptrace.Status_DEPRECATED_STATUS_CODE_UNKNOWN_ERROR,
		},
	}

	for _, test := range tests {
		t.Run(test.sendCode.String()+"/"+test.sendDeprecatedCode.String(), func(t *testing.T) {
			req := &otlpcollectortrace.ExportTraceServiceRequest{
				ResourceSpans: []*otlptrace.ResourceSpans{
					{
						InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
							{
								Spans: []*otlptrace.Span{
									{
										Status: otlptrace.Status{
											Code:           test.sendCode,
											DeprecatedCode: test.sendDeprecatedCode,
										},
									},
								},
							},
						},
					},
				},
			}

			TracesCompatibilityChanges(req)
			spanProto := req.ResourceSpans[0].InstrumentationLibrarySpans[0].Spans[0]
			// Check that DeprecatedCode is passed as is.
			assert.EqualValues(t, test.expectedRcvCode, spanProto.Status.Code)
			assert.EqualValues(t, test.expectedDeprecatedCode, spanProto.Status.DeprecatedCode)
		})
	}
}

func TestDeprecatedIntHistogram(t *testing.T) {
	tests := []struct {
		inputMetrics  []*otlpmetrics.Metric
		outputMetrics []*otlpmetrics.Metric
	}{
		{
			inputMetrics: []*otlpmetrics.Metric{
				{
					Data: &otlpmetrics.Metric_Histogram{
						Histogram: &otlpmetrics.Histogram{
							AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
							DataPoints: []*otlpmetrics.HistogramDataPoint{
								{
									Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
										{Key: "key", Value: "value"},
									},
									BucketCounts:      []uint64{11, 16, 2},
									ExplicitBounds:    []float64{3, 4},
									Sum:               10.1,
									StartTimeUnixNano: 0,
									TimeUnixNano:      1,
									Count:             29,
									Exemplars:         []otlpmetrics.Exemplar{},
								},
							},
						},
					},
				}},
			outputMetrics: []*otlpmetrics.Metric{{
				Data: &otlpmetrics.Metric_Histogram{
					Histogram: &otlpmetrics.Histogram{
						AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
						DataPoints: []*otlpmetrics.HistogramDataPoint{
							{
								Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
									{Key: "key", Value: "value"},
								},
								BucketCounts:      []uint64{11, 16, 2},
								ExplicitBounds:    []float64{3, 4},
								Sum:               10.1,
								StartTimeUnixNano: 0,
								TimeUnixNano:      1,
								Count:             29,
								Exemplars:         []otlpmetrics.Exemplar{},
								Attributes: []otlpcommon.KeyValue{
									{Key: "key", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value"}}},
								},
							},
						},
					},
				},
			}},
		},
		{
			inputMetrics: []*otlpmetrics.Metric{{
				Data: &otlpmetrics.Metric_IntHistogram{
					IntHistogram: &otlpmetrics.IntHistogram{ //nolint:staticcheck // SA1019 ignore this!
						AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
						DataPoints: []*otlpmetrics.IntHistogramDataPoint{ //nolint:staticcheck // SA1019 ignore this!
							{
								Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
									{Key: "key2", Value: "value2"},
								},
								BucketCounts:      []uint64{10, 15, 1},
								ExplicitBounds:    []float64{1, 2},
								Sum:               10,
								StartTimeUnixNano: 2,
								TimeUnixNano:      3,
								Count:             26,
								Exemplars:         []otlpmetrics.IntExemplar{}, //nolint:staticcheck // SA1019 ignore this!
							},
						},
					},
				},
			}},
			outputMetrics: []*otlpmetrics.Metric{{
				Data: &otlpmetrics.Metric_Histogram{
					Histogram: &otlpmetrics.Histogram{
						AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
						DataPoints: []*otlpmetrics.HistogramDataPoint{
							{
								Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
									{Key: "key2", Value: "value2"},
								},
								BucketCounts:      []uint64{10, 15, 1},
								ExplicitBounds:    []float64{1, 2},
								Sum:               10.0,
								StartTimeUnixNano: 2,
								TimeUnixNano:      3,
								Count:             26,
								Exemplars:         []otlpmetrics.Exemplar{},
								Attributes: []otlpcommon.KeyValue{
									{Key: "key2", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value2"}}},
								},
							},
						},
					},
				},
			}},
		},
		{
			inputMetrics: []*otlpmetrics.Metric{{
				Data: &otlpmetrics.Metric_IntHistogram{
					IntHistogram: &otlpmetrics.IntHistogram{ //nolint:staticcheck // SA1019 ignore this!
						AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
						DataPoints: []*otlpmetrics.IntHistogramDataPoint{ //nolint:staticcheck // SA1019 ignore this!
							{
								Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
									{Key: "key2", Value: "value2"},
								},
								BucketCounts:      []uint64{10, 15, 1},
								ExplicitBounds:    []float64{1, 2},
								Sum:               10,
								StartTimeUnixNano: 2,
								TimeUnixNano:      3,
								Count:             26,
								Exemplars:         []otlpmetrics.IntExemplar{}, //nolint:staticcheck // SA1019 ignore this!
							},
						},
					},
				},
			}},
			outputMetrics: []*otlpmetrics.Metric{{
				Data: &otlpmetrics.Metric_Histogram{
					Histogram: &otlpmetrics.Histogram{
						AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
						DataPoints: []*otlpmetrics.HistogramDataPoint{
							{
								Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
									{Key: "key2", Value: "value2"},
								},
								BucketCounts:      []uint64{10, 15, 1},
								ExplicitBounds:    []float64{1, 2},
								Sum:               10.0,
								StartTimeUnixNano: 2,
								TimeUnixNano:      3,
								Count:             26,
								Exemplars:         []otlpmetrics.Exemplar{},
								Attributes: []otlpcommon.KeyValue{
									{Key: "key2", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value2"}}},
								},
							},
						},
					},
				},
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.inputMetrics[0].Description, func(t *testing.T) {
			req := &otlpcollectormetrics.ExportMetricsServiceRequest{
				ResourceMetrics: []*otlpmetrics.ResourceMetrics{
					{
						InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
							{
								Metrics: test.inputMetrics,
							},
						}},
				},
			}
			MetricsCompatibilityChanges(req)
			assert.EqualValues(t, test.outputMetrics, req.ResourceMetrics[0].InstrumentationLibraryMetrics[0].Metrics)
		})
	}
}

func TestDeprecatedIntGauge(t *testing.T) {
	tests := []struct {
		inputMetrics  []*otlpmetrics.Metric
		outputMetrics []*otlpmetrics.Metric
	}{
		{
			inputMetrics: []*otlpmetrics.Metric{{
				Data: &otlpmetrics.Metric_Gauge{
					Gauge: &otlpmetrics.Gauge{
						DataPoints: []*otlpmetrics.NumberDataPoint{
							{
								Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
									{Key: "GaugeKey", Value: "GaugeValue"},
								},
								StartTimeUnixNano: 10,
								TimeUnixNano:      11,
								Value:             &otlpmetrics.NumberDataPoint_AsInt{AsInt: 100},
								Exemplars:         []otlpmetrics.Exemplar{},
							},
						},
					},
				},
			}},
			outputMetrics: []*otlpmetrics.Metric{{
				Data: &otlpmetrics.Metric_Gauge{
					Gauge: &otlpmetrics.Gauge{
						DataPoints: []*otlpmetrics.NumberDataPoint{
							{
								Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
									{Key: "GaugeKey", Value: "GaugeValue"},
								},
								StartTimeUnixNano: 10,
								TimeUnixNano:      11,
								Value:             &otlpmetrics.NumberDataPoint_AsInt{AsInt: 100},
								Exemplars:         []otlpmetrics.Exemplar{},
								Attributes: []otlpcommon.KeyValue{
									{Key: "GaugeKey", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "GaugeValue"}}},
								},
							},
						},
					},
				},
			}},
		},
		{
			inputMetrics: []*otlpmetrics.Metric{{
				Data: &otlpmetrics.Metric_IntGauge{
					IntGauge: &otlpmetrics.IntGauge{ //nolint:staticcheck // SA1019 ignore this!
						DataPoints: []*otlpmetrics.IntDataPoint{ //nolint:staticcheck // SA1019 ignore this!
							{
								Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
									{Key: "IntGaugeKey", Value: "IntGaugeValue"},
								},
								StartTimeUnixNano: 12,
								TimeUnixNano:      13,
								Value:             101,
								Exemplars:         []otlpmetrics.IntExemplar{}, //nolint:staticcheck // SA1019 ignore this!
							},
						},
					},
				},
			}},
			outputMetrics: []*otlpmetrics.Metric{{
				Data: &otlpmetrics.Metric_Gauge{
					Gauge: &otlpmetrics.Gauge{
						DataPoints: []*otlpmetrics.NumberDataPoint{
							{
								Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
									{Key: "IntGaugeKey", Value: "IntGaugeValue"},
								},
								StartTimeUnixNano: 12,
								TimeUnixNano:      13,
								Value:             &otlpmetrics.NumberDataPoint_AsInt{AsInt: 101},
								Exemplars:         []otlpmetrics.Exemplar{},
								Attributes: []otlpcommon.KeyValue{
									{Key: "IntGaugeKey", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "IntGaugeValue"}}},
								},
							},
						},
					},
				},
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.inputMetrics[0].Description, func(t *testing.T) {
			req := &otlpcollectormetrics.ExportMetricsServiceRequest{
				ResourceMetrics: []*otlpmetrics.ResourceMetrics{
					{
						InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
							{
								Metrics: test.inputMetrics,
							},
						}},
				},
			}
			MetricsCompatibilityChanges(req)
			assert.EqualValues(t, test.outputMetrics, req.ResourceMetrics[0].InstrumentationLibraryMetrics[0].Metrics)
		})
	}
}

func TestDeprecatedIntSum(t *testing.T) {
	tests := []struct {
		inputMetrics  []*otlpmetrics.Metric
		outputMetrics []*otlpmetrics.Metric
	}{
		{
			inputMetrics: []*otlpmetrics.Metric{{
				Data: &otlpmetrics.Metric_Sum{
					Sum: &otlpmetrics.Sum{
						AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
						IsMonotonic:            true,
						DataPoints: []*otlpmetrics.NumberDataPoint{
							{
								Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
									{Key: "SumKey", Value: "SumValue"},
								},
								StartTimeUnixNano: 20,
								TimeUnixNano:      21,
								Value:             &otlpmetrics.NumberDataPoint_AsInt{AsInt: 200},
								Exemplars: []otlpmetrics.Exemplar{
									{
										FilteredLabels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
											{Key: "FilteredLabelKey", Value: "FilteredLabelValue"},
										},
									},
								},
							},
						},
					},
				},
			}},
			outputMetrics: []*otlpmetrics.Metric{{
				Data: &otlpmetrics.Metric_Sum{
					Sum: &otlpmetrics.Sum{
						AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
						IsMonotonic:            true,
						DataPoints: []*otlpmetrics.NumberDataPoint{
							{
								Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
									{Key: "SumKey", Value: "SumValue"},
								},
								StartTimeUnixNano: 20,
								TimeUnixNano:      21,
								Value:             &otlpmetrics.NumberDataPoint_AsInt{AsInt: 200},
								Exemplars: []otlpmetrics.Exemplar{
									{
										FilteredLabels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
											{Key: "FilteredLabelKey", Value: "FilteredLabelValue"},
										},
										FilteredAttributes: []otlpcommon.KeyValue{ //nolint:staticcheck // SA1019 ignore this!
											{Key: "FilteredLabelKey", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "FilteredLabelValue"}}},
										},
									},
								},
								Attributes: []otlpcommon.KeyValue{
									{Key: "SumKey", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "SumValue"}}},
								},
							},
						},
					},
				},
			}},
		},
		{
			inputMetrics: []*otlpmetrics.Metric{{
				Data: &otlpmetrics.Metric_IntSum{
					IntSum: &otlpmetrics.IntSum{ //nolint:staticcheck // SA1019 ignore this!
						AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
						IsMonotonic:            true,
						DataPoints: []*otlpmetrics.IntDataPoint{ //nolint:staticcheck // SA1019 ignore this!
							{
								Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
									{Key: "IntSumKey", Value: "IntSumValue"},
								},
								StartTimeUnixNano: 22,
								TimeUnixNano:      23,
								Value:             201,
								Exemplars:         []otlpmetrics.IntExemplar{}, //nolint:staticcheck // SA1019 ignore this!
							},
						},
					},
				},
			}},
			outputMetrics: []*otlpmetrics.Metric{{
				Data: &otlpmetrics.Metric_Sum{
					Sum: &otlpmetrics.Sum{
						AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
						IsMonotonic:            true,
						DataPoints: []*otlpmetrics.NumberDataPoint{
							{
								Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
									{Key: "IntSumKey", Value: "IntSumValue"},
								},
								StartTimeUnixNano: 22,
								TimeUnixNano:      23,
								Value:             &otlpmetrics.NumberDataPoint_AsInt{AsInt: 201},
								Exemplars:         []otlpmetrics.Exemplar{},
								Attributes: []otlpcommon.KeyValue{
									{Key: "IntSumKey", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "IntSumValue"}}},
								},
							},
						},
					},
				},
			}},
		},
		{
			inputMetrics: []*otlpmetrics.Metric{{
				Data: &otlpmetrics.Metric_IntSum{
					IntSum: &otlpmetrics.IntSum{ //nolint:staticcheck // SA1019 ignore this!
						AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
						IsMonotonic:            false,
						DataPoints: []*otlpmetrics.IntDataPoint{ //nolint:staticcheck // SA1019 ignore this!
							{
								Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
									{Key: "IntSumKey", Value: "IntSumValue"},
								},
								StartTimeUnixNano: 22,
								TimeUnixNano:      23,
								Value:             201,
								Exemplars:         []otlpmetrics.IntExemplar{}, //nolint:staticcheck // SA1019 ignore this!
							},
						},
					},
				},
			}},
			outputMetrics: []*otlpmetrics.Metric{{
				Data: &otlpmetrics.Metric_Sum{
					Sum: &otlpmetrics.Sum{
						AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
						IsMonotonic:            false,
						DataPoints: []*otlpmetrics.NumberDataPoint{
							{
								Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
									{Key: "IntSumKey", Value: "IntSumValue"},
								},
								StartTimeUnixNano: 22,
								TimeUnixNano:      23,
								Value:             &otlpmetrics.NumberDataPoint_AsInt{AsInt: 201},
								Exemplars:         []otlpmetrics.Exemplar{},
								Attributes: []otlpcommon.KeyValue{
									{Key: "IntSumKey", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "IntSumValue"}}},
								},
							},
						},
					},
				},
			}},
		},
		{
			inputMetrics: []*otlpmetrics.Metric{{
				Data: &otlpmetrics.Metric_IntSum{
					IntSum: &otlpmetrics.IntSum{ //nolint:staticcheck // SA1019 ignore this!
						DataPoints: []*otlpmetrics.IntDataPoint{ //nolint:staticcheck // SA1019 ignore this!
							{
								Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
									{Key: "IntSumKey", Value: "IntSumValue"},
								},
								StartTimeUnixNano: 22,
								TimeUnixNano:      23,
								Value:             201,
								Exemplars: []otlpmetrics.IntExemplar{ //nolint:staticcheck // SA1019 ignore this!
									{
										FilteredLabels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
											{Key: "FilteredLabelKey", Value: "FilteredLabelValue"},
										},
									},
								},
							},
						},
					},
				},
			}},
			outputMetrics: []*otlpmetrics.Metric{{
				Data: &otlpmetrics.Metric_Sum{
					Sum: &otlpmetrics.Sum{
						DataPoints: []*otlpmetrics.NumberDataPoint{
							{
								Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
									{Key: "IntSumKey", Value: "IntSumValue"},
								},
								StartTimeUnixNano: 22,
								TimeUnixNano:      23,
								Value:             &otlpmetrics.NumberDataPoint_AsInt{AsInt: 201},
								Exemplars: []otlpmetrics.Exemplar{
									{
										Value: &otlpmetrics.Exemplar_AsInt{
											AsInt: 0,
										},
										FilteredLabels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
											{Key: "FilteredLabelKey", Value: "FilteredLabelValue"},
										},
										FilteredAttributes: []otlpcommon.KeyValue{ //nolint:staticcheck // SA1019 ignore this!
											{Key: "FilteredLabelKey", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "FilteredLabelValue"}}},
										},
									},
								},
								Attributes: []otlpcommon.KeyValue{
									{Key: "IntSumKey", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "IntSumValue"}}},
								},
							},
						},
					},
				},
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.inputMetrics[0].Description, func(t *testing.T) {
			req := &otlpcollectormetrics.ExportMetricsServiceRequest{
				ResourceMetrics: []*otlpmetrics.ResourceMetrics{
					{
						InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
							{
								Metrics: test.inputMetrics,
							},
						}},
				},
			}
			MetricsCompatibilityChanges(req)
			assert.EqualValues(t, test.outputMetrics, req.ResourceMetrics[0].InstrumentationLibraryMetrics[0].Metrics)
		})
	}
}

func TestAttributesAndLabels(t *testing.T) {
	tests := []struct {
		inputMetrics  []*otlpmetrics.Metric
		outputMetrics []*otlpmetrics.Metric
	}{
		{
			inputMetrics: []*otlpmetrics.Metric{
				{
					// only Labels are set
					Data: &otlpmetrics.Metric_Histogram{
						Histogram: &otlpmetrics.Histogram{
							AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
							DataPoints: []*otlpmetrics.HistogramDataPoint{
								{
									Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
										{Key: "key", Value: "value"},
									},
								},
							},
						},
					},
				},
			},
			outputMetrics: []*otlpmetrics.Metric{
				{
					Data: &otlpmetrics.Metric_Histogram{
						Histogram: &otlpmetrics.Histogram{
							AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
							DataPoints: []*otlpmetrics.HistogramDataPoint{
								{
									Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
										{Key: "key", Value: "value"},
									},
									Attributes: []otlpcommon.KeyValue{
										{Key: "key", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value"}}},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			inputMetrics: []*otlpmetrics.Metric{
				{
					// only Attributes are set
					Data: &otlpmetrics.Metric_Histogram{
						Histogram: &otlpmetrics.Histogram{
							AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
							DataPoints: []*otlpmetrics.HistogramDataPoint{
								{
									Attributes: []otlpcommon.KeyValue{
										{Key: "key2", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value2"}}},
									},
								},
							},
						},
					},
				},
			},
			outputMetrics: []*otlpmetrics.Metric{
				{
					Data: &otlpmetrics.Metric_Histogram{
						Histogram: &otlpmetrics.Histogram{
							AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
							DataPoints: []*otlpmetrics.HistogramDataPoint{
								{
									Attributes: []otlpcommon.KeyValue{
										{Key: "key2", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value2"}}},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			inputMetrics: []*otlpmetrics.Metric{
				{
					// both Attributes and Labels are set
					Data: &otlpmetrics.Metric_Histogram{
						Histogram: &otlpmetrics.Histogram{
							AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
							DataPoints: []*otlpmetrics.HistogramDataPoint{
								{
									Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
										{Key: "key1", Value: "value1"},
									},
									Attributes: []otlpcommon.KeyValue{
										{Key: "key2", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value2"}}},
									},
								},
							},
						},
					},
				},
			},
			outputMetrics: []*otlpmetrics.Metric{
				{
					Data: &otlpmetrics.Metric_Histogram{
						Histogram: &otlpmetrics.Histogram{
							AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
							DataPoints: []*otlpmetrics.HistogramDataPoint{
								{
									Labels: []otlpcommon.StringKeyValue{ //nolint:staticcheck // SA1019 ignore this!
										{Key: "key1", Value: "value1"},
									},
									Attributes: []otlpcommon.KeyValue{
										{Key: "key2", Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value2"}}},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.inputMetrics[0].Description, func(t *testing.T) {
			req := &otlpcollectormetrics.ExportMetricsServiceRequest{
				ResourceMetrics: []*otlpmetrics.ResourceMetrics{
					{
						InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
							{
								Metrics: test.inputMetrics,
							},
						}},
				},
			}
			MetricsCompatibilityChanges(req)
			assert.EqualValues(t, test.outputMetrics, req.ResourceMetrics[0].InstrumentationLibraryMetrics[0].Metrics)
		})
	}
}
