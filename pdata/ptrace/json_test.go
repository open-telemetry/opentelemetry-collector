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

package ptrace

import (
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"

	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

var tracesOTLP = func() Traces {
	td := NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().UpsertString("host.name", "testHost")
	il := rs.ScopeSpans().AppendEmpty()
	il.Scope().SetName("name")
	il.Scope().SetVersion("version")
	il.Spans().AppendEmpty().SetName("testSpan")
	return td
}()

var tracesJSON = `{"resourceSpans":[{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"testHost"}}]},"scopeSpans":[{"scope":{"name":"name","version":"version"},"spans":[{"traceId":"","spanId":"","parentSpanId":"","name":"testSpan","status":{}}]}]}]}`

func TestTracesJSON(t *testing.T) {
	encoder := NewJSONMarshaler()
	jsonBuf, err := encoder.MarshalTraces(tracesOTLP)
	assert.NoError(t, err)

	decoder := NewJSONUnmarshaler()
	var got interface{}
	got, err = decoder.UnmarshalTraces(jsonBuf)
	assert.NoError(t, err)

	assert.EqualValues(t, tracesOTLP, got)
}

func TestTracesJSON_Marshal(t *testing.T) {
	encoder := NewJSONMarshaler()
	jsonBuf, err := encoder.MarshalTraces(tracesOTLP)
	assert.NoError(t, err)
	assert.Equal(t, tracesJSON, string(jsonBuf))
}

var tracesOTLPFull = func() Traces {
	traceID := pcommon.TraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	spanID := pcommon.SpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
	td := NewTraces()
	// Add ResourceSpans.
	rs := td.ResourceSpans().AppendEmpty()
	rs.SetSchemaUrl("schemaURL")
	// Add resource.
	rs.Resource().Attributes().UpsertString("host.name", "testHost")
	rs.Resource().Attributes().UpsertString("service.name", "testService")
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
	sp.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	sp.SetTraceID(traceID)
	sp.SetSpanID(spanID)
	sp.SetDroppedEventsCount(1)
	sp.SetDroppedLinksCount(1)
	sp.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	sp.SetParentSpanID(spanID)
	sp.SetTraceState("state")
	sp.Status().SetCode(StatusCodeOk)
	sp.Status().SetMessage("message")
	// Add attributes.
	sp.Attributes().UpsertString("string", "value")
	sp.Attributes().UpsertBool("bool", true)
	sp.Attributes().UpsertInt("int", 1)
	sp.Attributes().UpsertDouble("double", 1.1)
	sp.Attributes().UpsertBytes("bytes", pcommon.NewImmutableByteSlice([]byte("foo")))
	arr := sp.Attributes().UpsertEmptySlice("array")
	arr.AppendEmpty().SetIntVal(1)
	arr.AppendEmpty().SetStringVal("str")
	kvList := sp.Attributes().UpsertEmptyMap("kvList")
	kvList.UpsertInt("int", 1)
	kvList.UpsertString("string", "string")
	// Add events.
	event := sp.Events().AppendEmpty()
	event.SetName("eventName")
	event.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	event.SetDroppedAttributesCount(1)
	event.Attributes().UpsertString("string", "value")
	event.Attributes().UpsertBool("bool", true)
	event.Attributes().UpsertInt("int", 1)
	event.Attributes().UpsertDouble("double", 1.1)
	event.Attributes().UpsertBytes("bytes", pcommon.NewImmutableByteSlice([]byte("foo")))
	// Add links.
	link := sp.Links().AppendEmpty()
	link.SetTraceState("state")
	link.SetTraceID(traceID)
	link.SetSpanID(spanID)
	link.SetDroppedAttributesCount(1)
	link.Attributes().UpsertString("string", "value")
	link.Attributes().UpsertBool("bool", true)
	link.Attributes().UpsertInt("int", 1)
	link.Attributes().UpsertDouble("double", 1.1)
	link.Attributes().UpsertBytes("bytes", pcommon.NewImmutableByteSlice([]byte("foo")))
	// Add another span.
	sp2 := il.Spans().AppendEmpty()
	sp2.SetName("testSpan2")
	return td
}()

func TestJSONFull(t *testing.T) {
	encoder := NewJSONMarshaler()
	jsonBuf, err := encoder.MarshalTraces(tracesOTLPFull)
	assert.NoError(t, err)

	decoder := NewJSONUnmarshaler()
	got, err := decoder.UnmarshalTraces(jsonBuf)
	assert.NoError(t, err)
	assert.EqualValues(t, tracesOTLPFull, got)
}

func BenchmarkJSONUnmarshal(b *testing.B) {
	b.ReportAllocs()

	encoder := NewJSONMarshaler()
	jsonBuf, err := encoder.MarshalTraces(tracesOTLPFull)
	assert.NoError(b, err)
	decoder := NewJSONUnmarshaler()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := decoder.UnmarshalTraces(jsonBuf)
			assert.NoError(b, err)
		}
	})
}

func TestReadTraceDataUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := readTraceData(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, otlptrace.TracesData{}, val)
}

func TestReadResourceSpansUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := readResourceSpans(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, &otlptrace.ResourceSpans{}, val)
}

func TestReadResourceSpansUnknownResourceField(t *testing.T) {
	jsonStr := `{"resource":{"extra":""}}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := readResourceSpans(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, &otlptrace.ResourceSpans{}, val)
}

func TestReadScopeSpansUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := readScopeSpans(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, &otlptrace.ScopeSpans{}, val)
}

func TestReadSpanUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := readSpan(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, &otlptrace.Span{}, val)
}

func TestReadSpanUnknownStatusField(t *testing.T) {
	jsonStr := `{"status":{"extra":""}}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := readSpan(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, &otlptrace.Span{}, val)
}

func TestReadSpanInvalidTraceIDField(t *testing.T) {
	jsonStr := `{"trace_id":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readSpan(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse trace_id")
	}
}

func TestReadSpanInvalidSpanIDField(t *testing.T) {
	jsonStr := `{"span_id":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readSpan(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse span_id")
	}
}

func TestReadSpanInvalidParentSpanIDField(t *testing.T) {
	jsonStr := `{"parent_span_id":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readSpan(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse parent_span_id")
	}
}

func TestReadSpanLinkUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := readSpanLink(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, &otlptrace.Span_Link{}, val)
}

func TestReadSpanLinkInvalidTraceIDField(t *testing.T) {
	jsonStr := `{"trace_id":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readSpanLink(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse trace_id")
	}
}

func TestReadSpanLinkInvalidSpanIDField(t *testing.T) {
	jsonStr := `{"span_id":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	readSpanLink(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse span_id")
	}
}

func TestReadSpanEventUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := readSpanEvent(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, &otlptrace.Span_Event{}, val)
}
