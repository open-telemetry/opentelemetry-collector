// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptrace

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestSpanCount(t *testing.T) {
	traces := NewTraces()
	assert.Equal(t, 0, traces.SpanCount())

	rs := traces.ResourceSpans().AppendEmpty()
	assert.Equal(t, 0, traces.SpanCount())

	ils := rs.ScopeSpans().AppendEmpty()
	assert.Equal(t, 0, traces.SpanCount())

	ils.Spans().AppendEmpty()
	assert.Equal(t, 1, traces.SpanCount())

	rms := traces.ResourceSpans()
	rms.EnsureCapacity(3)
	rms.AppendEmpty().ScopeSpans().AppendEmpty()
	ilss := rms.AppendEmpty().ScopeSpans().AppendEmpty().Spans()
	for range 5 {
		ilss.AppendEmpty()
	}
	// 5 + 1 (from rms.At(0) initialized first)
	assert.Equal(t, 6, traces.SpanCount())
}

func TestSpanCountWithEmpty(t *testing.T) {
	assert.Equal(t, 0, newTraces(&internal.ExportTraceServiceRequest{
		ResourceSpans: []*internal.ResourceSpans{{}},
	}, new(internal.State)).SpanCount())
	assert.Equal(t, 0, newTraces(&internal.ExportTraceServiceRequest{
		ResourceSpans: []*internal.ResourceSpans{
			{
				ScopeSpans: []*internal.ScopeSpans{{}},
			},
		},
	}, new(internal.State)).SpanCount())
	assert.Equal(t, 1, newTraces(&internal.ExportTraceServiceRequest{
		ResourceSpans: []*internal.ResourceSpans{
			{
				ScopeSpans: []*internal.ScopeSpans{
					{
						Spans: []*internal.Span{{}},
					},
				},
			},
		},
	}, new(internal.State)).SpanCount())
}

func TestTracesCopyTo(t *testing.T) {
	td := generateTestTraces()
	tracesCopy := NewTraces()
	td.CopyTo(tracesCopy)
	assert.Equal(t, td, tracesCopy)
}

func TestReadOnlyTracesInvalidUsage(t *testing.T) {
	td := NewTraces()
	assert.False(t, td.IsReadOnly())
	res := td.ResourceSpans().AppendEmpty().Resource()
	res.Attributes().PutStr("k1", "v1")
	td.MarkReadOnly()
	assert.True(t, td.IsReadOnly())
	assert.Panics(t, func() { res.Attributes().PutStr("k2", "v2") })
}

func BenchmarkTracesUsage(b *testing.B) {
	td := generateTestTraces()
	ts := pcommon.NewTimestampFromTime(time.Now())

	b.ReportAllocs()

	for b.Loop() {
		for i := 0; i < td.ResourceSpans().Len(); i++ {
			rs := td.ResourceSpans().At(i)
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

func BenchmarkTracesMarshalJSON(b *testing.B) {
	td := generateTestTraces()
	encoder := &JSONMarshaler{}

	b.ReportAllocs()

	for b.Loop() {
		jsonBuf, err := encoder.MarshalTraces(td)
		require.NoError(b, err)
		require.NotNil(b, jsonBuf)
	}
}
