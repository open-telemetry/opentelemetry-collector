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

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

var tracesOTLP = func() Traces {
	td := NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("host.name", "testHost")
	il := rs.ScopeSpans().AppendEmpty()
	il.Scope().SetName("name")
	il.Scope().SetVersion("version")
	il.Spans().AppendEmpty().SetName("testSpan")
	return td
}()

var tracesJSON = `{"resourceSpans":[{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"testHost"}}]},"scopeSpans":[{"scope":{"name":"name","version":"version"},"spans":[{"traceId":"","spanId":"","parentSpanId":"","name":"testSpan","status":{}}]}]}]}`

func TestTracesJSON(t *testing.T) {
	encoder := &JSONMarshaler{}
	jsonBuf, err := encoder.MarshalTraces(tracesOTLP)
	assert.NoError(t, err)

	decoder := &JSONUnmarshaler{}
	got, err := decoder.UnmarshalTraces(jsonBuf)
	assert.NoError(t, err)

	assert.EqualValues(t, tracesOTLP, got)
}

func TestTracesJSON_Marshal(t *testing.T) {
	encoder := &JSONMarshaler{}
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
	sp.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	sp.SetTraceID(traceID)
	sp.SetSpanID(spanID)
	sp.SetDroppedEventsCount(1)
	sp.SetDroppedLinksCount(1)
	sp.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now()))
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
	event.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
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

func TestJSONFull(t *testing.T) {
	encoder := &JSONMarshaler{}
	jsonBuf, err := encoder.MarshalTraces(tracesOTLPFull)
	assert.NoError(t, err)

	decoder := &JSONUnmarshaler{}
	got, err := decoder.UnmarshalTraces(jsonBuf)
	assert.NoError(t, err)
	assert.EqualValues(t, tracesOTLPFull, got)
}

func BenchmarkJSONUnmarshal(b *testing.B) {
	b.ReportAllocs()

	encoder := &JSONMarshaler{}
	jsonBuf, err := encoder.MarshalTraces(tracesOTLPFull)
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
