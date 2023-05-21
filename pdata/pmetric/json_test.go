// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetric

import (
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"

	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

var metricsOTLP = func() Metrics {
	md := NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("host.name", "testHost")
	il := rm.ScopeMetrics().AppendEmpty()
	il.Scope().SetName("name")
	il.Scope().SetVersion("version")
	il.Metrics().AppendEmpty().SetName("testMetric")
	return md
}()

var metricsJSON = `{"resourceMetrics":[{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"testHost"}}]},"scopeMetrics":[{"scope":{"name":"name","version":"version"},"metrics":[{"name":"testMetric"}]}]}]}`

func TestMetricsJSON(t *testing.T) {
	encoder := &JSONMarshaler{}
	jsonBuf, err := encoder.MarshalMetrics(metricsOTLP)
	assert.NoError(t, err)

	decoder := &JSONUnmarshaler{}
	got, err := decoder.UnmarshalMetrics(jsonBuf)
	assert.NoError(t, err)

	assert.EqualValues(t, metricsOTLP, got)
}

func TestMetricsJSON_Marshal(t *testing.T) {
	encoder := &JSONMarshaler{}
	jsonBuf, err := encoder.MarshalMetrics(metricsOTLP)
	assert.NoError(t, err)
	assert.Equal(t, metricsJSON, string(jsonBuf))
}

var metricsSumOTLPFull = func() Metrics {
	metric := NewMetrics()
	rs := metric.ResourceMetrics().AppendEmpty()
	rs.SetSchemaUrl("schemaURL")
	// Add resource.
	rs.Resource().Attributes().PutStr("host.name", "testHost")
	rs.Resource().Attributes().PutStr("service.name", "testService")
	rs.Resource().SetDroppedAttributesCount(1)
	// Add InstrumentationLibraryMetrics.
	m := rs.ScopeMetrics().AppendEmpty()
	m.Scope().SetName("instrumentation name")
	m.Scope().SetVersion("instrumentation version")
	m.Scope().Attributes().PutStr("instrumentation.attribute", "test")
	m.SetSchemaUrl("schemaURL")
	// Add Metric
	sumMetric := m.Metrics().AppendEmpty()
	sumMetric.SetName("test sum")
	sumMetric.SetDescription("test sum")
	sumMetric.SetUnit("unit")
	sum := sumMetric.SetEmptySum()
	sum.SetAggregationTemporality(AggregationTemporalityCumulative)
	sum.SetIsMonotonic(true)
	datapoint := sum.DataPoints().AppendEmpty()
	datapoint.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	datapoint.SetIntValue(100)
	datapoint.Attributes().PutStr("string", "val")
	datapoint.Attributes().PutBool("bool", true)
	datapoint.Attributes().PutInt("int", 1)
	datapoint.Attributes().PutDouble("double", 1.1)
	datapoint.Attributes().PutEmptyBytes("bytes").FromRaw([]byte("foo"))
	exemplar := datapoint.Exemplars().AppendEmpty()
	exemplar.SetDoubleValue(99.3)
	exemplar.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	traceID := pcommon.TraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	spanID := pcommon.SpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
	exemplar.SetSpanID(spanID)
	exemplar.SetTraceID(traceID)
	exemplar.FilteredAttributes().PutStr("service.name", "testService")
	datapoint.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	return metric
}

var metricsGaugeOTLPFull = func() Metrics {
	metric := NewMetrics()
	rs := metric.ResourceMetrics().AppendEmpty()
	rs.SetSchemaUrl("schemaURL")
	// Add resource.
	rs.Resource().Attributes().PutStr("host.name", "testHost")
	rs.Resource().Attributes().PutStr("service.name", "testService")
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
	datapoint.SetDoubleValue(10.2)
	datapoint.Attributes().PutStr("string", "val")
	datapoint.Attributes().PutBool("bool", true)
	datapoint.Attributes().PutInt("int", 1)
	datapoint.Attributes().PutDouble("double", 1.1)
	datapoint.Attributes().PutEmptyBytes("bytes").FromRaw([]byte("foo"))
	exemplar := datapoint.Exemplars().AppendEmpty()
	exemplar.SetDoubleValue(99.3)
	exemplar.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	traceID := pcommon.TraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	spanID := pcommon.SpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
	exemplar.SetSpanID(spanID)
	exemplar.SetTraceID(traceID)
	exemplar.FilteredAttributes().PutStr("service.name", "testService")
	datapoint.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	return metric
}

var metricsHistogramOTLPFull = func() Metrics {
	metric := NewMetrics()
	rs := metric.ResourceMetrics().AppendEmpty()
	rs.SetSchemaUrl("schemaURL")
	// Add resource.
	rs.Resource().Attributes().PutStr("host.name", "testHost")
	rs.Resource().Attributes().PutStr("service.name", "testService")
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
	histogram.SetAggregationTemporality(AggregationTemporalityCumulative)
	datapoint := histogram.DataPoints().AppendEmpty()
	datapoint.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	datapoint.Attributes().PutStr("string", "val")
	datapoint.Attributes().PutBool("bool", true)
	datapoint.Attributes().PutInt("int", 1)
	datapoint.Attributes().PutDouble("double", 1.1)
	datapoint.Attributes().PutEmptyBytes("bytes").FromRaw([]byte("foo"))
	datapoint.SetCount(4)
	datapoint.SetSum(345)
	datapoint.BucketCounts().FromRaw([]uint64{1, 1, 2})
	datapoint.ExplicitBounds().FromRaw([]float64{10, 100})
	exemplar := datapoint.Exemplars().AppendEmpty()
	exemplar.SetDoubleValue(99.3)
	exemplar.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	datapoint.SetMin(float64(time.Now().Unix()))
	traceID := pcommon.TraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	spanID := pcommon.SpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
	exemplar.SetSpanID(spanID)
	exemplar.SetTraceID(traceID)
	exemplar.FilteredAttributes().PutStr("service.name", "testService")
	datapoint.SetMax(float64(time.Now().Unix()))
	datapoint.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	return metric
}

var metricsExponentialHistogramOTLPFull = func() Metrics {
	metric := NewMetrics()
	rs := metric.ResourceMetrics().AppendEmpty()
	rs.SetSchemaUrl("schemaURL")
	// Add resource.
	rs.Resource().Attributes().PutStr("host.name", "testHost")
	rs.Resource().Attributes().PutStr("service.name", "testService")
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
	histogram.SetAggregationTemporality(AggregationTemporalityCumulative)
	datapoint := histogram.DataPoints().AppendEmpty()
	datapoint.SetScale(1)
	datapoint.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	datapoint.Attributes().PutStr("string", "val")
	datapoint.Attributes().PutBool("bool", true)
	datapoint.Attributes().PutInt("int", 1)
	datapoint.Attributes().PutDouble("double", 1.1)
	datapoint.Attributes().PutEmptyBytes("bytes").FromRaw([]byte("foo"))
	datapoint.SetCount(4)
	datapoint.SetSum(345)
	datapoint.Positive().BucketCounts().FromRaw([]uint64{1, 1, 2})
	datapoint.Positive().SetOffset(2)
	exemplar := datapoint.Exemplars().AppendEmpty()
	exemplar.SetDoubleValue(99.3)
	exemplar.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	datapoint.SetMin(float64(time.Now().Unix()))
	traceID := pcommon.TraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	spanID := pcommon.SpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
	exemplar.SetSpanID(spanID)
	exemplar.SetTraceID(traceID)
	exemplar.FilteredAttributes().PutStr("service.name", "testService")
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
	rs.Resource().Attributes().PutStr("host.name", "testHost")
	rs.Resource().Attributes().PutStr("service.name", "testService")
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
	datapoint.Attributes().PutStr("string", "val")
	datapoint.Attributes().PutBool("bool", true)
	datapoint.Attributes().PutInt("int", 1)
	datapoint.Attributes().PutDouble("double", 1.1)
	datapoint.Attributes().PutEmptyBytes("bytes").FromRaw([]byte("foo"))
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
			encoder := &JSONMarshaler{}
			m := tt.args.md()
			jsonBuf, err := encoder.MarshalMetrics(m)
			assert.NoError(t, err)
			decoder := JSONUnmarshaler{}
			got, err := decoder.UnmarshalMetrics(jsonBuf)
			assert.NoError(t, err)
			assert.EqualValues(t, m, got)
		})
	}
}

func TestUnmarshalJsoniterMetricsData(t *testing.T) {
	jsonStr := `{"extra":"", "resourceMetrics": []}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewMetrics()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, NewMetrics(), val)
}

func TestUnmarshalJsoniterResourceMetrics(t *testing.T) {
	jsonStr := `{"extra":"", "resource": {}, "schemaUrl": "schema", "scopeMetrics": []}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewResourceMetrics()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, &otlpmetrics.ResourceMetrics{SchemaUrl: "schema"}, val.orig)
}

func TestUnmarshalJsoniterScopeMetrics(t *testing.T) {
	jsonStr := `{"extra":"", "scope": {}, "metrics": [], "schemaUrl": "schema"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewScopeMetrics()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, &otlpmetrics.ScopeMetrics{SchemaUrl: "schema"}, val.orig)
}

func TestUnmarshalJsoniterMetric(t *testing.T) {
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
				jsonStr: `{"sum":{"extra":""}}`,
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
				jsonStr: `{"gauge":{"extra":""}}`,
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
				jsonStr: `{"histogram":{"extra":""}}`,
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
				jsonStr: `{"exponential_histogram":{"extra":""}}`,
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
				jsonStr: `{"summary":{"extra":""}}`,
			},
		},
		{
			name: "Metrics has unknown field",
			args: args{
				want:    &otlpmetrics.Metric{},
				jsonStr: `{"extra":""}`,
			},
		},
	}
	for _, tt := range tests {
		iter := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.args.jsonStr))
		jsoniter.ConfigFastest.ReturnIterator(iter)
		val := NewMetric()
		val.unmarshalJsoniter(iter)
		assert.NoError(t, iter.Error)
		assert.EqualValues(t, tt.args.want, val.orig)
	}
}

func TestUnmarshalJsoniterNumberDataPoint(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewNumberDataPoint()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, NewNumberDataPoint(), val)
}

func TestUnmarshalJsoniterHistogramDataPoint(t *testing.T) {
	jsonStr := `{"extra":"", "count":3}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewHistogramDataPoint()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, &otlpmetrics.HistogramDataPoint{Count: 3}, val.orig)
}

func TestUnmarshalJsoniterExponentialHistogramDataPoint(t *testing.T) {
	jsonStr := `{"extra":"", "count":3}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewExponentialHistogramDataPoint()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, &otlpmetrics.ExponentialHistogramDataPoint{Count: 3}, val.orig)
}

func TestUnmarshalJsoniterExponentialHistogramDataPointBuckets(t *testing.T) {
	jsonStr := `{"extra":"", "offset":3, "bucketCounts": [1, 2]}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewExponentialHistogramDataPointBuckets()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, &otlpmetrics.ExponentialHistogramDataPoint_Buckets{Offset: 3, BucketCounts: []uint64{1, 2}}, val.orig)
}

func TestUnmarshalJsoniterSummaryDataPoint(t *testing.T) {
	jsonStr := `{"extra":"", "count":3, "sum": 3.14}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewSummaryDataPoint()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, &otlpmetrics.SummaryDataPoint{
		Count: 3,
		Sum:   3.14,
	}, val.orig)
}

func TestUnmarshalJsoniterQuantileValue(t *testing.T) {
	jsonStr := `{"extra":"", "quantile":0.314, "value":3}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewSummaryDataPointValueAtQuantile()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, &otlpmetrics.SummaryDataPoint_ValueAtQuantile{
		Quantile: 0.314,
		Value:    3,
	}, val.orig)
}

func TestExemplarVal(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    *otlpmetrics.Exemplar
	}{
		{
			name:    "int",
			jsonStr: `{"asInt":1}`,
			want: &otlpmetrics.Exemplar{
				Value: &otlpmetrics.Exemplar_AsInt{
					AsInt: 1,
				},
			},
		},
		{
			name:    "double",
			jsonStr: `{"asDouble":3.14}`,
			want: &otlpmetrics.Exemplar{
				Value: &otlpmetrics.Exemplar_AsDouble{
					AsDouble: 3.14,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.jsonStr))
			defer jsoniter.ConfigFastest.ReturnIterator(iter)
			val := NewExemplar()
			val.unmarshalJsoniter(iter)
			assert.EqualValues(t, tt.want, val.orig)
		})
	}
}

func TestExemplarInvalidTraceID(t *testing.T) {
	jsonStr := `{"traceId":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	NewExemplar().unmarshalJsoniter(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse trace_id")
	}
}

func TestExemplarInvalidSpanID(t *testing.T) {
	jsonStr := `{"spanId":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	NewExemplar().unmarshalJsoniter(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse span_id")
	}
}

func TestExemplar(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewExemplar()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, NewExemplar(), val)
}
