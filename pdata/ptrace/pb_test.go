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
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/internal"
)

func TestProtoEncoding(t *testing.T) {
	tracesOTLPFull := generateTraces(2)
	encoder := NewProtoMarshaler()
	jsonBuf, err := encoder.MarshalTraces(tracesOTLPFull)
	assert.NoError(t, err)

	decoder := NewProtoUnmarshaler()
	got, err := decoder.UnmarshalTraces(jsonBuf)
	assert.NoError(t, err)
	assert.Equal(t, tracesOTLPFull, got)
}

func TestProtoTracesUnmarshaler_error(t *testing.T) {
	p := NewProtoUnmarshaler()
	_, err := p.UnmarshalTraces([]byte("+$%"))
	assert.Error(t, err)
}

func TestProtoSizer(t *testing.T) {
	sizer := NewProtoMarshaler().(Sizer)
	marshaler := NewProtoMarshaler()
	td := NewTraces()
	rms := td.ResourceSpans()
	rms.AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("foo")

	size := sizer.TracesSize(td)

	bytes, err := marshaler.MarshalTraces(td)
	require.NoError(t, err)
	assert.Equal(t, len(bytes), size)
}

func TestProtoSizer_withNil(t *testing.T) {
	sizer := NewProtoMarshaler().(Sizer)

	assert.Equal(t, 0, sizer.TracesSize(NewTraces()))
}

func BenchmarkTracesToProto(b *testing.B) {
	marshaler := NewProtoMarshaler()
	traces := generateBenchmarkTraces()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		buf, err := marshaler.MarshalTraces(traces)
		require.NoError(b, err)
		assert.NotEqual(b, 0, len(buf))
	}
}

func BenchmarkTracesFromProto(b *testing.B) {
	marshaler := NewProtoMarshaler()
	unmarshaler := NewProtoUnmarshaler()
	baseTraces := generateBenchmarkTraces()
	buf, err := marshaler.MarshalTraces(baseTraces)
	require.NoError(b, err)
	assert.NotEqual(b, 0, len(buf))
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		traces, err := unmarshaler.UnmarshalTraces(buf)
		require.NoError(b, err)
		assert.Equal(b, baseTraces.ResourceSpans().Len(), traces.ResourceSpans().Len())
	}
}

func generateBenchmarkTraces() Traces {
	return generateTraces(64)
}

func generateTraces(spansCount int) Traces {
	now := time.Now()

	traceID := internal.NewTraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	spanID := internal.NewSpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
	td := NewTraces()
	// Add ResourceSpans.
	rs := td.ResourceSpans().AppendEmpty()
	rs.SetSchemaUrl("schemaURL")
	// Add resource.
	rs.Resource().Attributes().UpsertString("service.name", "testService")
	rs.Resource().Attributes().UpsertInt("service.instance", 1234)
	rs.Resource().SetDroppedAttributesCount(1)

	// Add ScopeSpans.
	il := rs.ScopeSpans().AppendEmpty()
	il.SetSchemaUrl("schemaURL")
	// Add scope.
	il.Scope().SetName("scope name")
	il.Scope().SetVersion("scope version")
	il.Scope().Attributes().UpsertString("short.name", "scope")
	il.Scope().Attributes().UpsertDouble("short.version", 1.24)
	il.Scope().SetDroppedAttributesCount(2)

	// Add spans.
	il.Spans().EnsureCapacity(spansCount)
	for i := 0; i < spansCount; i++ {
		sp := il.Spans().AppendEmpty()
		sp.SetName("testSpan")
		sp.SetKind(internal.SpanKindClient)
		sp.SetDroppedAttributesCount(1)
		sp.SetStartTimestamp(internal.NewTimestampFromTime(now))
		sp.SetTraceID(traceID)
		sp.SetSpanID(spanID)
		sp.SetDroppedEventsCount(1)
		sp.SetDroppedLinksCount(1)
		sp.SetEndTimestamp(internal.NewTimestampFromTime(now.Add(5 * time.Millisecond)))
		sp.SetParentSpanID(spanID)
		sp.SetTraceState("state")
		sp.Status().SetCode(internal.StatusCodeOk)
		sp.Status().SetMessage("message")
		// Add attributes.
		sp.Attributes().UpsertString("string", "value")
		sp.Attributes().UpsertBool("bool", true)
		sp.Attributes().UpsertInt("int", 1)
		sp.Attributes().UpsertDouble("double", 1.1)
		sp.Attributes().UpsertBytes("bytes", internal.NewImmutableByteSlice([]byte("foo")))
		arr := internal.NewValueSlice()
		arr.SliceVal().AppendEmpty().SetIntVal(1)
		arr.SliceVal().AppendEmpty().SetStringVal("str")
		sp.Attributes().Upsert("array", arr)
		kvList := internal.NewValueMap()
		kvList.MapVal().Upsert("int", internal.NewValueInt(1))
		kvList.MapVal().Upsert("string", internal.NewValueString("string"))
		sp.Attributes().Upsert("kvList", kvList)
		// Add events.
		event := sp.Events().AppendEmpty()
		event.SetName("eventName")
		event.SetTimestamp(internal.NewTimestampFromTime(now.Add(2 * time.Millisecond)))
		event.SetDroppedAttributesCount(1)
		event.Attributes().UpsertString("string", "value")
		event.Attributes().UpsertBool("bool", true)
		event.Attributes().UpsertInt("int", 1)
		event.Attributes().UpsertDouble("double", 1.1)
		event.Attributes().UpsertBytes("bytes", internal.NewImmutableByteSlice([]byte("foo")))
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
		link.Attributes().UpsertBytes("bytes", internal.NewImmutableByteSlice([]byte("foo")))
	}
	return td
}
