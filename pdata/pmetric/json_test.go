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
	"github.com/stretchr/testify/assert"

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
	datapoint.Attributes().PutStr("string", "value")
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
	datapoint.Attributes().PutStr("string", "value")
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
	datapoint.Attributes().PutStr("string", "value")
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
	datapoint.Attributes().PutStr("string", "value")
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
	datapoint.Attributes().PutStr("string", "value")
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
	oldDelegate := delegate
	t.Cleanup(func() {
		delegate = oldDelegate
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, opEnumsAsInts := range []bool{true, false} {
				for _, opEmitDefaults := range []bool{true, false} {
					for _, opOrigName := range []bool{true, false} {
						delegate = &jsonpb.Marshaler{
							EnumsAsInts:  opEnumsAsInts,
							EmitDefaults: opEmitDefaults,
							OrigName:     opOrigName,
						}
						encoder := &JSONMarshaler{}
						m := tt.args.md()
						jsonBuf, err := encoder.MarshalMetrics(m)
						assert.NoError(t, err)
						decoder := JSONUnmarshaler{}
						got, err := decoder.UnmarshalMetrics(jsonBuf)
						assert.NoError(t, err)
						assert.EqualValues(t, m, got)
					}
				}
			}
		})
	}
}
