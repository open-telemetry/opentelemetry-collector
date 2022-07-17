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

package pmetric

import (
	"fmt"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
)

var metricsOTLP = func() Metrics {
	md := NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().UpsertString("host.name", "testHost")
	il := rm.ScopeMetrics().AppendEmpty()
	il.Scope().SetName("name")
	il.Scope().SetVersion("version")
	il.Metrics().AppendEmpty().SetName("testMetric")
	return md
}()

var metricsJSON = `{"resourceMetrics":[{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"testHost"}}]},"scopeMetrics":[{"scope":{"name":"name","version":"version"},"metrics":[{"name":"testMetric"}]}]}]}`

func TestMetricsJSON(t *testing.T) {
	encoder := NewJSONMarshaler()
	jsonBuf, err := encoder.MarshalMetrics(metricsOTLP)
	assert.NoError(t, err)

	decoder := NewJSONUnmarshaler()
	var got interface{}
	got, err = decoder.UnmarshalMetrics(jsonBuf)
	assert.NoError(t, err)

	assert.EqualValues(t, metricsOTLP, got)
}

func TestMetricsJSON_Marshal(t *testing.T) {
	encoder := NewJSONMarshaler()
	jsonBuf, err := encoder.MarshalMetrics(metricsOTLP)
	assert.NoError(t, err)
	assert.Equal(t, metricsJSON, string(jsonBuf))
}

func TestMetricsNil(t *testing.T) {
	jsonBuf := `{
"resourceMetrics": [
	{
	"resource": {
		"attributes": [
		{
			"key": "service.name",
			"value": {
			"stringValue": "unknown_service:node"
			}
		},
		{
			"key": "telemetry.sdk.language",
			"value": {
			"stringValue": "nodejs"
			}
		},
		{
			"key": "telemetry.sdk.name",
			"value": {
			"stringValue": "opentelemetry"
			}
		},
		{
			"key": "telemetry.sdk.version",
			"value": {
			"stringValue": "0.24.0"
			}
		}
		],
		"droppedAttributesCount": 0
	},
	"instrumentationLibraryMetrics": [
		{
		"metrics": [
			{
			"name": "metric_name",
			"description": "Example of a UpDownCounter",
			"unit": "1",
			"doubleSum": {
				"dataPoints": [
				{
					"labels": [
					{
						"key": "pid",
						"value": "50712"
					}
					],
					"value": 1,
					"startTimeUnixNano": 1631056185376000000,
					"timeUnixNano": 1631056185378763800
				}
				],
				"isMonotonic": false,
				"aggregationTemporality": 2
			}
			},
			{
			"name": "your_metric_name",
			"description": "Example of a sync observer with callback",
			"unit": "1",
			"doubleGauge": {
				"dataPoints": [
				{
					"labels": [
					{
						"key": "label",
						"value": "1"
					}
					],
					"value": 0.07604853280317792,
					"startTimeUnixNano": 1631056185376000000,
					"timeUnixNano": 1631056189394600700
				}
				]
			}
			},
			{
			"name": "your_metric_name",
			"description": "Example of a sync observer with callback",
			"unit": "1",
			"doubleGauge": {
				"dataPoints": [
				{
					"labels": [
					{
						"key": "label",
						"value": "2"
					}
					],
					"value": 0.9332005145656965,
					"startTimeUnixNano": 1631056185376000000,
					"timeUnixNano": 1631056189394630400
				}
				]
			}
			}
		],
		"instrumentationLibrary": {
			"name": "example-meter"
		}
		}
	]
	}
]
}`
	decoder := NewJSONUnmarshaler()
	var got interface{}
	got, err := decoder.UnmarshalMetrics([]byte(jsonBuf))
	assert.Error(t, err)
	assert.EqualValues(t, Metrics{}, got)
}

var metricsSumOTLPFull = func() Metrics {
	metric := NewMetrics()
	rs := metric.ResourceMetrics().AppendEmpty()
	rs.SetSchemaUrl("schemaURL")
	// Add resource.
	rs.Resource().Attributes().UpsertString("host.name", "testHost")
	rs.Resource().Attributes().UpsertString("service.name", "testService")
	rs.Resource().SetDroppedAttributesCount(1)
	// Add InstrumentationLibraryMetrics.
	m := rs.ScopeMetrics().AppendEmpty()
	m.Scope().SetName("instrumentation name")
	m.Scope().SetVersion("instrumentation version")
	m.SetSchemaUrl("schemaURL")
	// Add Metric
	sumData := m.Metrics().AppendEmpty()
	sumData.SetName("test sum")
	sumData.SetDescription("test sum")
	sumData.SetDataType(internal.MetricDataTypeSum)
	sumData.SetUnit("unit")
	sumData.Sum().SetAggregationTemporality(internal.MetricAggregationTemporalityCumulative)
	sumData.Sum().SetIsMonotonic(true)
	datapoint := sumData.Sum().DataPoints().AppendEmpty()
	datapoint.SetStartTimestamp(internal.NewTimestampFromTime(time.Now()))
	datapoint.SetFlags(internal.MetricDataPointFlags(otlpmetrics.DataPointFlags_FLAG_NO_RECORDED_VALUE))
	datapoint.SetIntVal(100)
	datapoint.Attributes().UpsertString("string", "value")
	datapoint.Attributes().UpsertBool("bool", true)
	datapoint.Attributes().UpsertInt("int", 1)
	datapoint.Attributes().UpsertDouble("double", 1.1)
	datapoint.Attributes().UpsertBytes("bytes", internal.NewImmutableByteSlice([]byte("foo")))
	exemplar := datapoint.Exemplars().AppendEmpty()
	exemplar.SetDoubleVal(99.3)
	exemplar.SetTimestamp(internal.NewTimestampFromTime(time.Now()))
	traceID := internal.NewTraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	spanID := internal.NewSpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
	exemplar.SetSpanID(spanID)
	exemplar.SetTraceID(traceID)
	exemplar.FilteredAttributes().UpsertString("service.name", "testService")
	datapoint.SetTimestamp(internal.NewTimestampFromTime(time.Now()))
	return metric
}

var metricsGaugeOTLPFull = func() Metrics {
	metric := NewMetrics()
	rs := metric.ResourceMetrics().AppendEmpty()
	rs.SetSchemaUrl("schemaURL")
	// Add resource.
	rs.Resource().Attributes().UpsertString("host.name", "testHost")
	rs.Resource().Attributes().UpsertString("service.name", "testService")
	rs.Resource().SetDroppedAttributesCount(1)
	// Add InstrumentationLibraryMetrics.
	m := rs.ScopeMetrics().AppendEmpty()
	m.Scope().SetName("instrumentation name")
	m.Scope().SetVersion("instrumentation version")
	m.SetSchemaUrl("schemaURL")
	// Add Metric
	gaugeData := m.Metrics().AppendEmpty()
	gaugeData.SetName("test gauge")
	gaugeData.SetDescription("test gauge")
	gaugeData.SetDataType(internal.MetricDataTypeGauge)
	gaugeData.SetUnit("unit")
	datapoint := gaugeData.Gauge().DataPoints().AppendEmpty()
	datapoint.SetStartTimestamp(internal.NewTimestampFromTime(time.Now()))
	datapoint.SetFlags(internal.MetricDataPointFlags(otlpmetrics.DataPointFlags_FLAG_NO_RECORDED_VALUE))
	datapoint.SetDoubleVal(10.2)
	datapoint.Attributes().UpsertString("string", "value")
	datapoint.Attributes().UpsertBool("bool", true)
	datapoint.Attributes().UpsertInt("int", 1)
	datapoint.Attributes().UpsertDouble("double", 1.1)
	datapoint.Attributes().UpsertBytes("bytes", internal.NewImmutableByteSlice([]byte("foo")))
	exemplar := datapoint.Exemplars().AppendEmpty()
	exemplar.SetDoubleVal(99.3)
	exemplar.SetTimestamp(internal.NewTimestampFromTime(time.Now()))
	traceID := internal.NewTraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	spanID := internal.NewSpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
	exemplar.SetSpanID(spanID)
	exemplar.SetTraceID(traceID)
	exemplar.FilteredAttributes().UpsertString("service.name", "testService")
	datapoint.SetTimestamp(internal.NewTimestampFromTime(time.Now()))
	return metric
}

var metricsHistogramOTLPFull = func() Metrics {
	metric := NewMetrics()
	rs := metric.ResourceMetrics().AppendEmpty()
	rs.SetSchemaUrl("schemaURL")
	// Add resource.
	rs.Resource().Attributes().UpsertString("host.name", "testHost")
	rs.Resource().Attributes().UpsertString("service.name", "testService")
	rs.Resource().SetDroppedAttributesCount(1)
	// Add InstrumentationLibraryMetrics.
	m := rs.ScopeMetrics().AppendEmpty()
	m.Scope().SetName("instrumentation name")
	m.Scope().SetVersion("instrumentation version")
	m.SetSchemaUrl("schemaURL")
	// Add Metric
	histogramData := m.Metrics().AppendEmpty()
	histogramData.SetName("test Histogram")
	histogramData.SetDescription("test Histogram")
	histogramData.SetDataType(internal.MetricDataTypeHistogram)
	histogramData.SetUnit("unit")
	histogramData.Histogram().SetAggregationTemporality(MetricAggregationTemporalityCumulative)
	datapoint := histogramData.Histogram().DataPoints().AppendEmpty()
	datapoint.SetStartTimestamp(internal.NewTimestampFromTime(time.Now()))
	datapoint.SetFlags(internal.MetricDataPointFlags(otlpmetrics.DataPointFlags_FLAG_NO_RECORDED_VALUE))
	datapoint.Attributes().UpsertString("string", "value")
	datapoint.Attributes().UpsertBool("bool", true)
	datapoint.Attributes().UpsertInt("int", 1)
	datapoint.Attributes().UpsertDouble("double", 1.1)
	datapoint.Attributes().UpsertBytes("bytes", internal.NewImmutableByteSlice([]byte("foo")))
	datapoint.SetCount(4)
	datapoint.SetSum(345)
	datapoint.SetBucketCounts(internal.NewImmutableUInt64Slice([]uint64{1, 1, 2}))
	datapoint.SetExplicitBounds(internal.NewImmutableFloat64Slice([]float64{10, 100}))
	exemplar := datapoint.Exemplars().AppendEmpty()
	exemplar.SetDoubleVal(99.3)
	exemplar.SetTimestamp(internal.NewTimestampFromTime(time.Now()))
	datapoint.SetMin(float64(time.Now().Unix()))
	traceID := internal.NewTraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	spanID := internal.NewSpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
	exemplar.SetSpanID(spanID)
	exemplar.SetTraceID(traceID)
	exemplar.FilteredAttributes().UpsertString("service.name", "testService")
	datapoint.SetMax(float64(time.Now().Unix()))
	datapoint.SetTimestamp(internal.NewTimestampFromTime(time.Now()))
	return metric
}

var metricsExponentialHistogramOTLPFull = func() Metrics {
	metric := NewMetrics()
	rs := metric.ResourceMetrics().AppendEmpty()
	rs.SetSchemaUrl("schemaURL")
	// Add resource.
	rs.Resource().Attributes().UpsertString("host.name", "testHost")
	rs.Resource().Attributes().UpsertString("service.name", "testService")
	rs.Resource().SetDroppedAttributesCount(1)
	// Add InstrumentationLibraryMetrics.
	m := rs.ScopeMetrics().AppendEmpty()
	m.Scope().SetName("instrumentation name")
	m.Scope().SetVersion("instrumentation version")
	m.SetSchemaUrl("schemaURL")
	// Add Metric
	histogramData := m.Metrics().AppendEmpty()
	histogramData.SetName("test ExponentialHistogram")
	histogramData.SetDescription("test ExponentialHistogram")
	histogramData.SetDataType(internal.MetricDataTypeExponentialHistogram)
	histogramData.SetUnit("unit")
	histogramData.ExponentialHistogram().SetAggregationTemporality(MetricAggregationTemporalityCumulative)
	datapoint := histogramData.ExponentialHistogram().DataPoints().AppendEmpty()
	datapoint.SetScale(1)
	datapoint.SetStartTimestamp(internal.NewTimestampFromTime(time.Now()))
	datapoint.SetFlags(internal.MetricDataPointFlags(otlpmetrics.DataPointFlags_FLAG_NO_RECORDED_VALUE))
	datapoint.Attributes().UpsertString("string", "value")
	datapoint.Attributes().UpsertBool("bool", true)
	datapoint.Attributes().UpsertInt("int", 1)
	datapoint.Attributes().UpsertDouble("double", 1.1)
	datapoint.Attributes().UpsertBytes("bytes", internal.NewImmutableByteSlice([]byte("foo")))
	datapoint.SetCount(4)
	datapoint.SetSum(345)
	datapoint.Positive().SetBucketCounts(internal.NewImmutableUInt64Slice([]uint64{1, 1, 2}))
	datapoint.Positive().SetOffset(2)
	exemplar := datapoint.Exemplars().AppendEmpty()
	exemplar.SetDoubleVal(99.3)
	exemplar.SetTimestamp(internal.NewTimestampFromTime(time.Now()))
	datapoint.SetMin(float64(time.Now().Unix()))
	traceID := internal.NewTraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	spanID := internal.NewSpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
	exemplar.SetSpanID(spanID)
	exemplar.SetTraceID(traceID)
	exemplar.FilteredAttributes().UpsertString("service.name", "testService")
	datapoint.SetMax(float64(time.Now().Unix()))
	datapoint.Negative().SetBucketCounts(internal.NewImmutableUInt64Slice([]uint64{1, 1, 2}))
	datapoint.Negative().SetOffset(2)
	datapoint.SetZeroCount(5)
	datapoint.SetTimestamp(internal.NewTimestampFromTime(time.Now()))
	return metric
}

var metricsSummaryOTLPFull = func() Metrics {
	metric := NewMetrics()
	rs := metric.ResourceMetrics().AppendEmpty()
	rs.SetSchemaUrl("schemaURL")
	// Add resource.
	rs.Resource().Attributes().UpsertString("host.name", "testHost")
	rs.Resource().Attributes().UpsertString("service.name", "testService")
	rs.Resource().SetDroppedAttributesCount(1)
	// Add InstrumentationLibraryMetrics.
	m := rs.ScopeMetrics().AppendEmpty()
	m.Scope().SetName("instrumentation name")
	m.Scope().SetVersion("instrumentation version")
	m.SetSchemaUrl("schemaURL")
	// Add Metric
	sumData := m.Metrics().AppendEmpty()
	sumData.SetName("test summary")
	sumData.SetDescription("test summary")
	sumData.SetDataType(internal.MetricDataTypeSummary)
	sumData.SetUnit("unit")
	datapoint := sumData.Summary().DataPoints().AppendEmpty()
	datapoint.SetStartTimestamp(internal.NewTimestampFromTime(time.Now()))
	datapoint.SetFlags(internal.MetricDataPointFlags(otlpmetrics.DataPointFlags_FLAG_NO_RECORDED_VALUE))
	datapoint.SetCount(100)
	datapoint.SetSum(100)
	quantile := datapoint.QuantileValues().AppendEmpty()
	quantile.SetQuantile(0.5)
	quantile.SetValue(1.2)
	datapoint.Attributes().UpsertString("string", "value")
	datapoint.Attributes().UpsertBool("bool", true)
	datapoint.Attributes().UpsertInt("int", 1)
	datapoint.Attributes().UpsertDouble("double", 1.1)
	datapoint.Attributes().UpsertBytes("bytes", internal.NewImmutableByteSlice([]byte("foo")))
	datapoint.SetTimestamp(internal.NewTimestampFromTime(time.Now()))
	return metric
}

func Test_jsonUnmarshaler_UnmarshalMetrics(t *testing.T) {
	type args struct {
		EnumsAsInts  bool
		EmitDefaults bool
		OrigName     bool
		md           func() Metrics
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "sum,default options",
			args: args{
				EnumsAsInts:  false,
				EmitDefaults: false,
				OrigName:     false,
				md:           metricsSumOTLPFull,
			},
		},
		{
			name: "sum, EnumsAsInts set true",
			args: args{
				EnumsAsInts:  true,
				EmitDefaults: false,
				OrigName:     false,
				md:           metricsSumOTLPFull,
			},
		},
		{
			name: "sum, EmitDefaults set true",
			args: args{
				EnumsAsInts:  false,
				EmitDefaults: true,
				OrigName:     false,
				md:           metricsSumOTLPFull,
			},
		},
		{
			name: "sum, OrigName set true",
			args: args{
				EnumsAsInts:  false,
				EmitDefaults: false,
				OrigName:     true,
				md:           metricsSumOTLPFull,
			},
		},
		{
			name: "sum, EmitDefaults and  OrigName set true",
			args: args{
				EnumsAsInts:  false,
				EmitDefaults: true,
				OrigName:     true,
				md:           metricsSumOTLPFull,
			},
		},
		{
			name: "sum, EnumsAsInts and OrigName options set true",
			args: args{
				EnumsAsInts:  true,
				EmitDefaults: false,
				OrigName:     true,
				md:           metricsSumOTLPFull,
			},
		},
		{
			name: "sum, EnumsAsInts AND EmitDefaults  set true",
			args: args{
				EnumsAsInts:  true,
				EmitDefaults: true,
				OrigName:     false,
				md:           metricsSumOTLPFull,
			},
		},
		{
			name: "sum, all options set true",
			args: args{
				EnumsAsInts:  true,
				EmitDefaults: true,
				OrigName:     true,
				md:           metricsSumOTLPFull,
			},
		},

		{
			name: "gauge,default options",
			args: args{
				EnumsAsInts:  false,
				EmitDefaults: false,
				OrigName:     false,
				md:           metricsGaugeOTLPFull,
			},
		},
		{
			name: "gauge, EnumsAsInts set true",
			args: args{
				EnumsAsInts:  true,
				EmitDefaults: false,
				OrigName:     false,
				md:           metricsGaugeOTLPFull,
			},
		},
		{
			name: "gauge, EmitDefaults set true",
			args: args{
				EnumsAsInts:  false,
				EmitDefaults: true,
				OrigName:     false,
				md:           metricsGaugeOTLPFull,
			},
		},
		{
			name: "gauge, OrigName set true",
			args: args{
				EnumsAsInts:  false,
				EmitDefaults: false,
				OrigName:     true,
				md:           metricsGaugeOTLPFull,
			},
		},
		{
			name: "gauge, EmitDefaults and  OrigName set true",
			args: args{
				EnumsAsInts:  false,
				EmitDefaults: true,
				OrigName:     true,
				md:           metricsGaugeOTLPFull,
			},
		},
		{
			name: "gauge, EnumsAsInts and OrigName options set true",
			args: args{
				EnumsAsInts:  true,
				EmitDefaults: false,
				OrigName:     true,
				md:           metricsGaugeOTLPFull,
			},
		},
		{
			name: "gauge, EnumsAsInts AND EmitDefaults  set true",
			args: args{
				EnumsAsInts:  true,
				EmitDefaults: true,
				OrigName:     false,
				md:           metricsGaugeOTLPFull,
			},
		},
		{
			name: "gauge, all options set true",
			args: args{
				EnumsAsInts:  true,
				EmitDefaults: true,
				OrigName:     true,
				md:           metricsGaugeOTLPFull,
			},
		},

		{
			name: "Histogram,default options",
			args: args{
				EnumsAsInts:  false,
				EmitDefaults: false,
				OrigName:     false,
				md:           metricsHistogramOTLPFull,
			},
		},
		{
			name: "Histogram, EnumsAsInts set true",
			args: args{
				EnumsAsInts:  true,
				EmitDefaults: false,
				OrigName:     false,
				md:           metricsHistogramOTLPFull,
			},
		},
		{
			name: "Histogram, EmitDefaults set true",
			args: args{
				EnumsAsInts:  false,
				EmitDefaults: true,
				OrigName:     false,
				md:           metricsHistogramOTLPFull,
			},
		},
		{
			name: "Histogram, OrigName set true",
			args: args{
				EnumsAsInts:  false,
				EmitDefaults: false,
				OrigName:     true,
				md:           metricsHistogramOTLPFull,
			},
		},
		{
			name: "Histogram, EmitDefaults and  OrigName set true",
			args: args{
				EnumsAsInts:  false,
				EmitDefaults: true,
				OrigName:     true,
				md:           metricsHistogramOTLPFull,
			},
		},
		{
			name: "Histogram, EnumsAsInts and OrigName options set true",
			args: args{
				EnumsAsInts:  true,
				EmitDefaults: false,
				OrigName:     true,
				md:           metricsHistogramOTLPFull,
			},
		},
		{
			name: "Histogram, EnumsAsInts AND EmitDefaults  set true",
			args: args{
				EnumsAsInts:  true,
				EmitDefaults: true,
				OrigName:     false,
				md:           metricsHistogramOTLPFull,
			},
		},
		{
			name: "Histogram, all options set true",
			args: args{
				EnumsAsInts:  true,
				EmitDefaults: true,
				OrigName:     true,
				md:           metricsHistogramOTLPFull,
			},
		},

		{
			name: "ExponentialHistogram,default options",
			args: args{
				EnumsAsInts:  false,
				EmitDefaults: false,
				OrigName:     false,
				md:           metricsExponentialHistogramOTLPFull,
			},
		},
		{
			name: "ExponentialHistogram, EnumsAsInts set true",
			args: args{
				EnumsAsInts:  true,
				EmitDefaults: false,
				OrigName:     false,
				md:           metricsExponentialHistogramOTLPFull,
			},
		},
		{
			name: "ExponentialHistogram, EmitDefaults set true",
			args: args{
				EnumsAsInts:  false,
				EmitDefaults: true,
				OrigName:     false,
				md:           metricsExponentialHistogramOTLPFull,
			},
		},
		{
			name: "ExponentialHistogram, OrigName set true",
			args: args{
				EnumsAsInts:  false,
				EmitDefaults: false,
				OrigName:     true,
				md:           metricsExponentialHistogramOTLPFull,
			},
		},
		{
			name: "ExponentialHistogram, EmitDefaults and  OrigName set true",
			args: args{
				EnumsAsInts:  false,
				EmitDefaults: true,
				OrigName:     true,
				md:           metricsExponentialHistogramOTLPFull,
			},
		},
		{
			name: "ExponentialHistogram, EnumsAsInts and OrigName options set true",
			args: args{
				EnumsAsInts:  true,
				EmitDefaults: false,
				OrigName:     true,
				md:           metricsExponentialHistogramOTLPFull,
			},
		},
		{
			name: "ExponentialHistogram, EnumsAsInts AND EmitDefaults  set true",
			args: args{
				EnumsAsInts:  true,
				EmitDefaults: true,
				OrigName:     false,
				md:           metricsExponentialHistogramOTLPFull,
			},
		},
		{
			name: "ExponentialHistogram, all options set true",
			args: args{
				EnumsAsInts:  true,
				EmitDefaults: true,
				OrigName:     true,
				md:           metricsExponentialHistogramOTLPFull,
			},
		},

		{
			name: "Summary,default options",
			args: args{
				EnumsAsInts:  false,
				EmitDefaults: false,
				OrigName:     false,
				md:           metricsSummaryOTLPFull,
			},
		},
		{
			name: "Summary, EnumsAsInts set true",
			args: args{
				EnumsAsInts:  true,
				EmitDefaults: false,
				OrigName:     false,
				md:           metricsSummaryOTLPFull,
			},
		},
		{
			name: "Summary, EmitDefaults set true",
			args: args{
				EnumsAsInts:  false,
				EmitDefaults: true,
				OrigName:     false,
				md:           metricsSummaryOTLPFull,
			},
		},
		{
			name: "Summary, OrigName set true",
			args: args{
				EnumsAsInts:  false,
				EmitDefaults: false,
				OrigName:     true,
				md:           metricsSummaryOTLPFull,
			},
		},
		{
			name: "Summary, EmitDefaults and  OrigName set true",
			args: args{
				EnumsAsInts:  false,
				EmitDefaults: true,
				OrigName:     true,
				md:           metricsSummaryOTLPFull,
			},
		},
		{
			name: "Summary, EnumsAsInts and OrigName options set true",
			args: args{
				EnumsAsInts:  true,
				EmitDefaults: false,
				OrigName:     true,
				md:           metricsSummaryOTLPFull,
			},
		},
		{
			name: "Summary, EnumsAsInts AND EmitDefaults  set true",
			args: args{
				EnumsAsInts:  true,
				EmitDefaults: true,
				OrigName:     false,
				md:           metricsSummaryOTLPFull,
			},
		},
		{
			name: "Summary, all options set true",
			args: args{
				EnumsAsInts:  true,
				EmitDefaults: true,
				OrigName:     true,
				md:           metricsSummaryOTLPFull,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marshaller := &jsonMarshaler{
				delegate: jsonpb.Marshaler{
					EnumsAsInts:  tt.args.EnumsAsInts,
					EmitDefaults: tt.args.EmitDefaults,
					OrigName:     tt.args.OrigName,
				}}
			m := tt.args.md()
			jsonBuf, err := marshaller.MarshalMetrics(m)
			assert.NoError(t, err)
			decoder := NewJSONUnmarshaler()
			got, err := decoder.UnmarshalMetrics(jsonBuf)
			assert.NoError(t, err)
			assert.EqualValues(t, m, got)
		})
	}
}

func TestReadAttributeUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readAttribute(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadAnyValueUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readAnyValue(iter, "")
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadAnyValueInvliadBytesValue(t *testing.T) {
	jsonStr := `"--"`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readAnyValue(iter, "bytesValue")
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "base64")
	}
}

func TestReadArrayUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readArray(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadKvlistValueUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readKvlistValue(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadAggregationTemporality(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    otlpmetrics.AggregationTemporality
	}{
		{
			name: "string",
			jsonStr: fmt.Sprintf(`"%s"`,
				otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE.String()),
			want: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
		},
		{
			name: "string",
			jsonStr: fmt.Sprintf(`"%s"`,
				otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA.String()),
			want: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
		},
		{
			name: "int",
			jsonStr: fmt.Sprintf("%d",
				otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE),
			want: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
		},
		{
			name: "int",
			jsonStr: fmt.Sprintf("%d",
				otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA),
			want: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.jsonStr))
			defer jsoniter.ConfigFastest.ReturnIterator(iter)
			if got := readAggregationTemporality(iter); got != tt.want {
				t.Errorf("readSpanKind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadArrayValueeInvalidArrayValue(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readAnyValue(iter, "arrayValue")
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadKvlistValueInvalidArrayValue(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readAnyValue(iter, "kvlistValue")
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadMetricsDataUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readMetricsData(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestExemplar_IntVal(t *testing.T) {
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
	readExemplar(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadArray(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    *otlpcommon.ArrayValue
	}{
		{
			name: "values",
			jsonStr: `{"values":[{
"stringValue":"12312"
}]}`,
			want: &otlpcommon.ArrayValue{
				Values: []otlpcommon.AnyValue{
					{
						Value: &otlpcommon.AnyValue_StringValue{
							StringValue: "12312",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.jsonStr))
			defer jsoniter.ConfigFastest.ReturnIterator(iter)
			got := readArray(iter)
			assert.EqualValues(t, tt.want, got)
		})
	}
}

func TestReadKvlistValue(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    *otlpcommon.KeyValueList
	}{
		{
			name: "values",
			jsonStr: `{"values":[{
"key":"testKey",
"value":{
"stringValue": "testValue"
}
}]}`,
			want: &otlpcommon.KeyValueList{
				Values: []otlpcommon.KeyValue{
					{
						Key: "testKey",
						Value: otlpcommon.AnyValue{
							Value: &otlpcommon.AnyValue_StringValue{
								StringValue: "testValue",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.jsonStr))
			defer jsoniter.ConfigFastest.ReturnIterator(iter)
			got := readKvlistValue(iter)
			assert.EqualValues(t, tt.want, got)
		})
	}
}

func TestReadResourceMetricsResourceUnknown(t *testing.T) {
	jsonStr := `{"resource":{"exists":"true"}}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readResourceMetrics(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}
func TestReadResourceMetricsUnknownField(t *testing.T) {
	jsonStr := `{"exists":"true"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readResourceMetrics(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadInstrumentationLibraryMetricsUnknownField(t *testing.T) {
	jsonStr := `{"exists":"true"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readInstrumentationLibraryMetrics(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
	}
}

func TestReadInstrumentationLibraryUnknownField(t *testing.T) {
	jsonStr := `{"instrumentationLibrary":{"exists":"true"}}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readInstrumentationLibraryMetrics(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "unknown field")
		assert.Contains(t, iter.Error.Error(), "instrumentationLibrary")
	}
}

func TestReadMetricUnknownField(t *testing.T) {
	type args struct {
		wantErrorMsg string
		jsonStr      string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "sum has unknown field",
			args: args{
				wantErrorMsg: "readMetric.Sum",
				jsonStr:      `{"sum":{"exists":"true"}}`,
			},
		},
		{
			name: "gauge has unknown field",
			args: args{
				wantErrorMsg: "readMetric.Gauge",
				jsonStr:      `{"gauge":{"exists":"true"}}`,
			},
		},
		{
			name: "histogram has unknown field",
			args: args{
				wantErrorMsg: "readMetric.Histogram",
				jsonStr:      `{"histogram":{"exists":"true"}}`,
			},
		},
		{
			name: "exponential_histogram has unknown field",
			args: args{
				wantErrorMsg: "readMetric.ExponentialHistogram",
				jsonStr:      `{"exponential_histogram":{"exists":"true"}}`,
			},
		},
		{
			name: "Summary has unknown field",
			args: args{
				wantErrorMsg: "readMetric.Summary",
				jsonStr:      `{"summary":{"exists":"true"}}`,
			},
		},
	}
	for _, tt := range tests {
		iter := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.args.jsonStr))
		jsoniter.ConfigFastest.ReturnIterator(iter)
		readMetric(iter)
		if assert.Error(t, iter.Error) {
			assert.Contains(t, iter.Error.Error(), "unknown field")
			assert.Contains(t, iter.Error.Error(), tt.args.wantErrorMsg)
		}
	}
}
