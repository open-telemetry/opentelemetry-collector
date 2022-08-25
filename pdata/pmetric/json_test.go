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
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"

	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
	"go.opentelemetry.io/collector/pdata/pcommon"
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
	m.Scope().Attributes().UpsertString("instrumentation.attribute", "test")
	m.SetSchemaUrl("schemaURL")
	// Add Metric
	sumMetric := m.Metrics().AppendEmpty()
	sumMetric.SetName("test sum")
	sumMetric.SetDescription("test sum")
	sumMetric.SetUnit("unit")
	sum := sumMetric.SetEmptySum()
	sum.SetAggregationTemporality(MetricAggregationTemporalityCumulative)
	sum.SetIsMonotonic(true)
	datapoint := sum.DataPoints().AppendEmpty()
	datapoint.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	datapoint.SetIntVal(100)
	datapoint.Attributes().UpsertString("string", "value")
	datapoint.Attributes().UpsertBool("bool", true)
	datapoint.Attributes().UpsertInt("int", 1)
	datapoint.Attributes().UpsertDouble("double", 1.1)
	datapoint.Attributes().UpsertEmptyBytes("bytes").FromRaw([]byte("foo"))
	exemplar := datapoint.Exemplars().AppendEmpty()
	exemplar.SetDoubleVal(99.3)
	exemplar.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	traceID := pcommon.TraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	spanID := pcommon.SpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
	exemplar.SetSpanID(spanID)
	exemplar.SetTraceID(traceID)
	exemplar.FilteredAttributes().UpsertString("service.name", "testService")
	datapoint.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
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
	gaugeMetric := m.Metrics().AppendEmpty()
	gaugeMetric.SetName("test gauge")
	gaugeMetric.SetDescription("test gauge")
	gaugeMetric.SetUnit("unit")
	gauge := gaugeMetric.SetEmptyGauge()
	datapoint := gauge.DataPoints().AppendEmpty()
	datapoint.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	datapoint.SetDoubleVal(10.2)
	datapoint.Attributes().UpsertString("string", "value")
	datapoint.Attributes().UpsertBool("bool", true)
	datapoint.Attributes().UpsertInt("int", 1)
	datapoint.Attributes().UpsertDouble("double", 1.1)
	datapoint.Attributes().UpsertEmptyBytes("bytes").FromRaw([]byte("foo"))
	exemplar := datapoint.Exemplars().AppendEmpty()
	exemplar.SetDoubleVal(99.3)
	exemplar.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	traceID := pcommon.TraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	spanID := pcommon.SpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
	exemplar.SetSpanID(spanID)
	exemplar.SetTraceID(traceID)
	exemplar.FilteredAttributes().UpsertString("service.name", "testService")
	datapoint.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
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
	histogramMetric := m.Metrics().AppendEmpty()
	histogramMetric.SetName("test Histogram")
	histogramMetric.SetDescription("test Histogram")
	histogramMetric.SetUnit("unit")
	histogram := histogramMetric.SetEmptyHistogram()
	histogram.SetAggregationTemporality(MetricAggregationTemporalityCumulative)
	datapoint := histogram.DataPoints().AppendEmpty()
	datapoint.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	datapoint.Attributes().UpsertString("string", "value")
	datapoint.Attributes().UpsertBool("bool", true)
	datapoint.Attributes().UpsertInt("int", 1)
	datapoint.Attributes().UpsertDouble("double", 1.1)
	datapoint.Attributes().UpsertEmptyBytes("bytes").FromRaw([]byte("foo"))
	datapoint.SetCount(4)
	datapoint.SetSum(345)
	datapoint.BucketCounts().FromRaw([]uint64{1, 1, 2})
	datapoint.ExplicitBounds().FromRaw([]float64{10, 100})
	exemplar := datapoint.Exemplars().AppendEmpty()
	exemplar.SetDoubleVal(99.3)
	exemplar.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	datapoint.SetMin(float64(time.Now().Unix()))
	traceID := pcommon.TraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	spanID := pcommon.SpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
	exemplar.SetSpanID(spanID)
	exemplar.SetTraceID(traceID)
	exemplar.FilteredAttributes().UpsertString("service.name", "testService")
	datapoint.SetMax(float64(time.Now().Unix()))
	datapoint.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
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
	histogramMetric := m.Metrics().AppendEmpty()
	histogramMetric.SetName("test ExponentialHistogram")
	histogramMetric.SetDescription("test ExponentialHistogram")
	histogramMetric.SetUnit("unit")
	histogram := histogramMetric.SetEmptyExponentialHistogram()
	histogram.SetAggregationTemporality(MetricAggregationTemporalityCumulative)
	datapoint := histogram.DataPoints().AppendEmpty()
	datapoint.SetScale(1)
	datapoint.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	datapoint.Attributes().UpsertString("string", "value")
	datapoint.Attributes().UpsertBool("bool", true)
	datapoint.Attributes().UpsertInt("int", 1)
	datapoint.Attributes().UpsertDouble("double", 1.1)
	datapoint.Attributes().UpsertEmptyBytes("bytes").FromRaw([]byte("foo"))
	datapoint.SetCount(4)
	datapoint.SetSum(345)
	datapoint.Positive().BucketCounts().FromRaw([]uint64{1, 1, 2})
	datapoint.Positive().SetOffset(2)
	exemplar := datapoint.Exemplars().AppendEmpty()
	exemplar.SetDoubleVal(99.3)
	exemplar.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	datapoint.SetMin(float64(time.Now().Unix()))
	traceID := pcommon.TraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	spanID := pcommon.SpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
	exemplar.SetSpanID(spanID)
	exemplar.SetTraceID(traceID)
	exemplar.FilteredAttributes().UpsertString("service.name", "testService")
	datapoint.SetMax(float64(time.Now().Unix()))
	datapoint.Negative().BucketCounts().FromRaw([]uint64{1, 1, 2})
	datapoint.Negative().SetOffset(2)
	datapoint.SetZeroCount(5)
	datapoint.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
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
	summaryMetric := m.Metrics().AppendEmpty()
	summaryMetric.SetName("test summary")
	summaryMetric.SetDescription("test summary")
	summaryMetric.SetUnit("unit")
	summary := summaryMetric.SetEmptySummary()
	datapoint := summary.DataPoints().AppendEmpty()
	datapoint.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	datapoint.SetCount(100)
	datapoint.SetSum(100)
	quantile := datapoint.QuantileValues().AppendEmpty()
	quantile.SetQuantile(0.5)
	quantile.SetValue(1.2)
	datapoint.Attributes().UpsertString("string", "value")
	datapoint.Attributes().UpsertBool("bool", true)
	datapoint.Attributes().UpsertInt("int", 1)
	datapoint.Attributes().UpsertDouble("double", 1.1)
	datapoint.Attributes().UpsertEmptyBytes("bytes").FromRaw([]byte("foo"))
	datapoint.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	return metric
}

func Test_jsonUnmarshaler_UnmarshalMetrics(t *testing.T) {
	type args struct {
		md func() Metrics
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "sum",
			args: args{
				md: metricsSumOTLPFull,
			},
		},
		{
			name: "gauge",
			args: args{
				md: metricsGaugeOTLPFull,
			},
		},
		{
			name: "Histogram",
			args: args{
				md: metricsHistogramOTLPFull,
			},
		},
		{
			name: "ExponentialHistogram",
			args: args{
				md: metricsExponentialHistogramOTLPFull,
			},
		},
		{
			name: "Summary",
			args: args{
				md: metricsSummaryOTLPFull,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, opEnumsAsInts := range []bool{true, false} {
				for _, opEmitDefaults := range []bool{true, false} {
					for _, opOrigName := range []bool{true, false} {
						marshaller := &jsonMarshaler{
							delegate: jsonpb.Marshaler{
								EnumsAsInts:  opEnumsAsInts,
								EmitDefaults: opEmitDefaults,
								OrigName:     opOrigName,
							}}
						m := tt.args.md()
						jsonBuf, err := marshaller.MarshalMetrics(m)
						assert.NoError(t, err)
						decoder := NewJSONUnmarshaler()
						got, err := decoder.UnmarshalMetrics(jsonBuf)
						assert.NoError(t, err)
						assert.EqualValues(t, m, got)
					}
				}
			}
		})
	}
}

func TestReadMetricsDataUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	value := readMetricsData(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, otlpmetrics.MetricsData{}, value)
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
