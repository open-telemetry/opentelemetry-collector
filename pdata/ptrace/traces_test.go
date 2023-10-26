// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptrace

import (
	"testing"
	"time"

	gogoproto "github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	goproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	otlpcollectortrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/trace/v1"
	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestSpanCount(t *testing.T) {
	traces := NewTraces()
	assert.EqualValues(t, 0, traces.SpanCount())

	rs := traces.ResourceSpans().AppendEmpty()
	assert.EqualValues(t, 0, traces.SpanCount())

	ils := rs.ScopeSpans().AppendEmpty()
	assert.EqualValues(t, 0, traces.SpanCount())

	ils.Spans().AppendEmpty()
	assert.EqualValues(t, 1, traces.SpanCount())

	rms := traces.ResourceSpans()
	rms.EnsureCapacity(3)
	rms.AppendEmpty().ScopeSpans().AppendEmpty()
	ilss := rms.AppendEmpty().ScopeSpans().AppendEmpty().Spans()
	for i := 0; i < 5; i++ {
		ilss.AppendEmpty()
	}
	// 5 + 1 (from rms.At(0) initialized first)
	assert.EqualValues(t, 6, traces.SpanCount())
}

func TestSpanCountWithEmpty(t *testing.T) {
	assert.EqualValues(t, 0, newTraces(&otlpcollectortrace.ExportTraceServiceRequest{
		ResourceSpans: []*otlptrace.ResourceSpans{{}},
	}).SpanCount())
	assert.EqualValues(t, 0, newTraces(&otlpcollectortrace.ExportTraceServiceRequest{
		ResourceSpans: []*otlptrace.ResourceSpans{
			{
				ScopeSpans: []*otlptrace.ScopeSpans{{}},
			},
		},
	}).SpanCount())
	assert.EqualValues(t, 1, newTraces(&otlpcollectortrace.ExportTraceServiceRequest{
		ResourceSpans: []*otlptrace.ResourceSpans{
			{
				ScopeSpans: []*otlptrace.ScopeSpans{
					{
						Spans: []*otlptrace.Span{{}},
					},
				},
			},
		},
	}).SpanCount())
}

func TestToFromOtlp(t *testing.T) {
	otlp := &otlpcollectortrace.ExportTraceServiceRequest{}
	traces := newTraces(otlp)
	assert.EqualValues(t, NewTraces(), traces)
	assert.EqualValues(t, otlp, traces.getOrig())
	// More tests in ./tracedata/traces_test.go. Cannot have them here because of
	// circular dependency.
}

func TestResourceSpansWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate ResourceSpans as pdata struct.
	traces := NewTraces()
	fillTestResourceSpansSlice(traces.ResourceSpans())

	// Marshal its underlying ProtoBuf to wire.
	wire1, err := gogoproto.Marshal(traces.getOrig())
	assert.NoError(t, err)
	assert.NotNil(t, wire1)

	// Unmarshal from the wire to OTLP Protobuf in goproto's representation.
	var goprotoMessage emptypb.Empty
	err = goproto.Unmarshal(wire1, &goprotoMessage)
	assert.NoError(t, err)

	// Marshal to the wire again.
	wire2, err := goproto.Marshal(&goprotoMessage)
	assert.NoError(t, err)
	assert.NotNil(t, wire2)

	// Unmarshal from the wire into gogoproto's representation.
	var gogoprotoRS2 otlpcollectortrace.ExportTraceServiceRequest
	err = gogoproto.Unmarshal(wire2, &gogoprotoRS2)
	assert.NoError(t, err)

	// Now compare that the original and final ProtoBuf messages are the same.
	// This proves that goproto and gogoproto marshaling/unmarshaling are wire compatible.
	assert.EqualValues(t, traces.getOrig(), &gogoprotoRS2)
}

func TestTracesCopyTo(t *testing.T) {
	traces := NewTraces()
	fillTestResourceSpansSlice(traces.ResourceSpans())
	tracesCopy := NewTraces()
	traces.CopyTo(tracesCopy)
	assert.EqualValues(t, traces, tracesCopy)
}

func TestReadOnlyTracesInvalidUsage(t *testing.T) {
	traces := NewTraces()
	assert.False(t, traces.IsReadOnly())
	res := traces.ResourceSpans().AppendEmpty().Resource()
	res.Attributes().PutStr("k1", "v1")
	traces.MarkReadOnly()
	assert.True(t, traces.IsReadOnly())
	assert.Panics(t, func() { res.Attributes().PutStr("k2", "v2") })
}

func BenchmarkTracesUsage(b *testing.B) {
	traces := NewTraces()
	fillTestResourceSpansSlice(traces.ResourceSpans())
	ts := pcommon.NewTimestampFromTime(time.Now())

	b.ReportAllocs()
	b.ResetTimer()

	for bb := 0; bb < b.N; bb++ {
		for i := 0; i < traces.ResourceSpans().Len(); i++ {
			rs := traces.ResourceSpans().At(i)
			res := rs.Resource()
			res.Attributes().PutStr("foo", "bar")
			v, ok := res.Attributes().Get("foo")
			assert.True(b, ok)
			assert.Equal(b, "bar", v.Str())
			v.SetStr("new-bar")
			assert.Equal(b, "new-bar", v.Str())
			res.Attributes().Remove("foo")
			for j := 0; j < rs.ScopeSpans().Len(); j++ {
				iss := rs.ScopeSpans().At(j)
				iss.Scope().SetName("new_test_name")
				assert.Equal(b, "new_test_name", iss.Scope().Name())
				for k := 0; k < iss.Spans().Len(); k++ {
					s := iss.Spans().At(k)
					s.SetName("new_span")
					assert.Equal(b, "new_span", s.Name())
					s.SetStartTimestamp(ts)
					assert.Equal(b, ts, s.StartTimestamp())
					s.SetEndTimestamp(ts)
					assert.Equal(b, ts, s.EndTimestamp())
					s.SetTraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
					assert.Equal(b, pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}), s.TraceID())
					s.SetSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8})
					assert.Equal(b, pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}), s.SpanID())
				}
				s := iss.Spans().AppendEmpty()
				s.SetName("another_span")
				s.SetStartTimestamp(ts)
				s.SetEndTimestamp(ts)
				s.SetTraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
				s.SetParentSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8})
				s.SetSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8})
				s.Attributes().PutStr("foo1", "bar1")
				s.Attributes().PutStr("foo2", "bar2")
				iss.Spans().RemoveIf(func(lr Span) bool {
					return lr.Name() == "another_span"
				})
			}
		}
	}
}
