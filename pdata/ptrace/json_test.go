// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptrace

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

var _ Marshaler = (*JSONMarshaler)(nil)
var _ Unmarshaler = (*JSONUnmarshaler)(nil)

var tracesOTLP = func() Traces {
	startTimestamp := pcommon.Timestamp(1684617382541971000)
	eventTimestamp := pcommon.Timestamp(1684620382541971000)
	endTimestamp := pcommon.Timestamp(1684623646539558000)
	traceID := pcommon.TraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	spanID := pcommon.SpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
	td := NewTraces()
	// Add ResourceSpans.
	rs := td.ResourceSpans().AppendEmpty()
	rs.SetSchemaUrl("schemaURL")
	// Add resource.
	rs.Resource().Attributes().PutStr("host.name", "testHost")
	rs.Resource().Attributes().PutStr("service.name", "testService")
	rs.Resource().SetDroppedAttributesCount(1)
	// Add ScopeSpans.
	il := rs.ScopeSpans().AppendEmpty()
	il.Scope().SetName("scope name")
	il.Scope().SetVersion("scope version")
	il.SetSchemaUrl("schemaURL")
	// Add spans.
	sp := il.Spans().AppendEmpty()
	sp.SetName("testSpan")
	sp.SetKind(SpanKindClient)
	sp.SetDroppedAttributesCount(1)
	sp.SetStartTimestamp(startTimestamp)
	sp.SetTraceID(traceID)
	sp.SetSpanID(spanID)
	sp.SetDroppedEventsCount(1)
	sp.SetDroppedLinksCount(1)
	sp.SetEndTimestamp(endTimestamp)
	sp.SetParentSpanID(spanID)
	sp.TraceState().FromRaw("state")
	sp.Status().SetCode(StatusCodeOk)
	sp.Status().SetMessage("message")
	// Add attributes.
	sp.Attributes().PutStr("string", "value")
	sp.Attributes().PutBool("bool", true)
	sp.Attributes().PutInt("int", 1)
	sp.Attributes().PutDouble("double", 1.1)
	sp.Attributes().PutEmptyBytes("bytes").FromRaw([]byte("foo"))
	arr := sp.Attributes().PutEmptySlice("array")
	arr.AppendEmpty().SetInt(1)
	arr.AppendEmpty().SetStr("str")
	kvList := sp.Attributes().PutEmptyMap("kvList")
	kvList.PutInt("int", 1)
	kvList.PutStr("string", "string")
	// Add events.
	event := sp.Events().AppendEmpty()
	event.SetName("eventName")
	event.SetTimestamp(eventTimestamp)
	event.SetDroppedAttributesCount(1)
	event.Attributes().PutStr("string", "value")
	event.Attributes().PutBool("bool", true)
	event.Attributes().PutInt("int", 1)
	event.Attributes().PutDouble("double", 1.1)
	event.Attributes().PutEmptyBytes("bytes").FromRaw([]byte("foo"))
	// Add links.
	link := sp.Links().AppendEmpty()
	link.TraceState().FromRaw("state")
	link.SetTraceID(traceID)
	link.SetSpanID(spanID)
	link.SetDroppedAttributesCount(1)
	link.Attributes().PutStr("string", "value")
	link.Attributes().PutBool("bool", true)
	link.Attributes().PutInt("int", 1)
	link.Attributes().PutDouble("double", 1.1)
	link.Attributes().PutEmptyBytes("bytes").FromRaw([]byte("foo"))
	// Add another span.
	sp2 := il.Spans().AppendEmpty()
	sp2.SetName("testSpan2")
	return td
}()

var tracesJSON = `{"resourceSpans":[{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"testHost"}},{"key":"service.name","value":{"stringValue":"testService"}}],"droppedAttributesCount":1},"scopeSpans":[{"scope":{"name":"scope name","version":"scope version"},"spans":[{"traceId":"0102030405060708090a0b0c0d0e0f10","spanId":"1112131415161718","traceState":"state","parentSpanId":"1112131415161718","name":"testSpan","kind":3,"startTimeUnixNano":"1684617382541971000","endTimeUnixNano":"1684623646539558000","attributes":[{"key":"string","value":{"stringValue":"value"}},{"key":"bool","value":{"boolValue":true}},{"key":"int","value":{"intValue":"1"}},{"key":"double","value":{"doubleValue":1.1}},{"key":"bytes","value":{"bytesValue":"Zm9v"}},{"key":"array","value":{"arrayValue":{"values":[{"intValue":"1"},{"stringValue":"str"}]}}},{"key":"kvList","value":{"kvlistValue":{"values":[{"key":"int","value":{"intValue":"1"}},{"key":"string","value":{"stringValue":"string"}}]}}}],"droppedAttributesCount":1,"events":[{"timeUnixNano":"1684620382541971000","name":"eventName","attributes":[{"key":"string","value":{"stringValue":"value"}},{"key":"bool","value":{"boolValue":true}},{"key":"int","value":{"intValue":"1"}},{"key":"double","value":{"doubleValue":1.1}},{"key":"bytes","value":{"bytesValue":"Zm9v"}}],"droppedAttributesCount":1}],"droppedEventsCount":1,"links":[{"traceId":"0102030405060708090a0b0c0d0e0f10","spanId":"1112131415161718","traceState":"state","attributes":[{"key":"string","value":{"stringValue":"value"}},{"key":"bool","value":{"boolValue":true}},{"key":"int","value":{"intValue":"1"}},{"key":"double","value":{"doubleValue":1.1}},{"key":"bytes","value":{"bytesValue":"Zm9v"}}],"droppedAttributesCount":1}],"droppedLinksCount":1,"status":{"message":"message","code":1}},{"traceId":"","spanId":"","parentSpanId":"","name":"testSpan2","status":{}}],"schemaUrl":"schemaURL"}],"schemaUrl":"schemaURL"}]}`

func TestJSONUnmarshal(t *testing.T) {
	decoder := &JSONUnmarshaler{}
	got, err := decoder.UnmarshalTraces([]byte(tracesJSON))
	assert.NoError(t, err)
	assert.EqualValues(t, tracesOTLP, got)
}

func TestJSONMarshal(t *testing.T) {
	encoder := &JSONMarshaler{}
	jsonBuf, err := encoder.MarshalTraces(tracesOTLP)
	assert.NoError(t, err)
	assert.Equal(t, tracesJSON, string(jsonBuf))
}

func TestJSONUnmarshalInvalid(t *testing.T) {
	jsonStr := `{"extra":"", "resourceSpans": "extra"}`
	decoder := &JSONUnmarshaler{}
	_, err := decoder.UnmarshalTraces([]byte(jsonStr))
	assert.Error(t, err)
}

func TestUnmarshalJsoniterTraceData(t *testing.T) {
	jsonStr := `{"extra":"", "resourceSpans": [{"extra":""}]}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewTraces()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, 1, val.ResourceSpans().Len())
}

func TestUnmarshalJsoniterResourceSpans(t *testing.T) {
	jsonStr := `{"extra":"", "resource": {}, "scopeSpans": []}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewResourceSpans()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, NewResourceSpans(), val)
}

func TestUnmarshalJsoniterScopeSpans(t *testing.T) {
	jsonStr := `{"extra":"", "scope": {}, "logRecords": []}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewScopeSpans()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, NewScopeSpans(), val)
}

func TestUnmarshalJsoniterSpan(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewSpan()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, NewSpan(), val)
}

func TestUnmarshalJsoniterSpanInvalidTraceIDField(t *testing.T) {
	jsonStr := `{"trace_id":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	NewSpan().unmarshalJsoniter(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse trace_id")
	}
}

func TestUnmarshalJsoniterSpanInvalidSpanIDField(t *testing.T) {
	jsonStr := `{"span_id":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	NewSpan().unmarshalJsoniter(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse span_id")
	}
}

func TestUnmarshalJsoniterSpanInvalidParentSpanIDField(t *testing.T) {
	jsonStr := `{"parent_span_id":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	NewSpan().unmarshalJsoniter(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse parent_span_id")
	}
}

func TestUnmarshalJsoniterSpanStatus(t *testing.T) {
	jsonStr := `{"status":{"extra":""}}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewStatus()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, NewStatus(), val)
}

func TestUnmarshalJsoniterSpanLink(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewSpanLink()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, NewSpanLink(), val)
}

func TestUnmarshalJsoniterSpanLinkInvalidTraceIDField(t *testing.T) {
	jsonStr := `{"trace_id":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	NewSpanLink().unmarshalJsoniter(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse trace_id")
	}
}

func TestUnmarshalJsoniterSpanLinkInvalidSpanIDField(t *testing.T) {
	jsonStr := `{"span_id":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	NewSpanLink().unmarshalJsoniter(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse span_id")
	}
}

func TestUnmarshalJsoniterSpanEvent(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewSpanEvent()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, NewSpanEvent(), val)
}

func BenchmarkJSONUnmarshal(b *testing.B) {
	b.ReportAllocs()

	encoder := &JSONMarshaler{}
	jsonBuf, err := encoder.MarshalTraces(tracesOTLP)
	assert.NoError(b, err)
	decoder := &JSONUnmarshaler{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := decoder.UnmarshalTraces(jsonBuf)
			assert.NoError(b, err)
		}
	})
}
