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

package pdata

import (
	"testing"

	gogoproto "github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	goproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	otlptrace "go.opentelemetry.io/collector/internal/data/protogen/trace/v1"
)

func TestSpanCount(t *testing.T) {
	md := NewTraces()
	assert.EqualValues(t, 0, md.SpanCount())

	md.ResourceSpans().Resize(1)
	assert.EqualValues(t, 0, md.SpanCount())

	md.ResourceSpans().At(0).InstrumentationLibrarySpans().Resize(1)
	assert.EqualValues(t, 0, md.SpanCount())

	md.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().Resize(1)
	assert.EqualValues(t, 1, md.SpanCount())

	rms := md.ResourceSpans()
	rms.Resize(3)
	rms.At(0).InstrumentationLibrarySpans().Resize(1)
	rms.At(0).InstrumentationLibrarySpans().At(0).Spans().Resize(1)
	rms.At(1).InstrumentationLibrarySpans().Resize(1)
	rms.At(2).InstrumentationLibrarySpans().Resize(1)
	rms.At(2).InstrumentationLibrarySpans().At(0).Spans().Resize(5)
	assert.EqualValues(t, 6, md.SpanCount())
}

func TestSize(t *testing.T) {
	md := NewTraces()
	assert.Equal(t, 0, md.Size())
	rms := md.ResourceSpans()
	rms.Resize(1)
	rms.At(0).InstrumentationLibrarySpans().Resize(1)
	rms.At(0).InstrumentationLibrarySpans().At(0).Spans().Resize(1)
	rms.At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).SetName("foo")
	otlp := TracesToOtlp(md)
	size := 0
	sizeBytes := 0
	for _, rspans := range otlp {
		size += rspans.Size()
		bts, err := rspans.Marshal()
		require.NoError(t, err)
		sizeBytes += len(bts)
	}
	assert.Equal(t, size, md.Size())
	assert.Equal(t, sizeBytes, md.Size())
}

func TestTracesSizeWithNil(t *testing.T) {
	assert.Equal(t, 0, TracesFromOtlp([]*otlptrace.ResourceSpans{nil}).Size())
}

func TestSpanCountWithEmpty(t *testing.T) {
	assert.EqualValues(t, 0, TracesFromOtlp([]*otlptrace.ResourceSpans{{}}).SpanCount())
	assert.EqualValues(t, 0, TracesFromOtlp([]*otlptrace.ResourceSpans{
		{
			InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{{}},
		},
	}).SpanCount())
	assert.EqualValues(t, 1, TracesFromOtlp([]*otlptrace.ResourceSpans{
		{
			InstrumentationLibrarySpans: []*otlptrace.InstrumentationLibrarySpans{
				{
					Spans: []*otlptrace.Span{{}},
				},
			},
		},
	}).SpanCount())
}

func TestSpanID(t *testing.T) {
	sid := InvalidSpanID()
	assert.EqualValues(t, [8]byte{}, sid.Bytes())
	assert.True(t, sid.IsEmpty())

	sid = NewSpanID([8]byte{1, 2, 3, 4, 4, 3, 2, 1})
	assert.EqualValues(t, [8]byte{1, 2, 3, 4, 4, 3, 2, 1}, sid.Bytes())
	assert.False(t, sid.IsEmpty())
}

func TestTraceID(t *testing.T) {
	tid := InvalidTraceID()
	assert.EqualValues(t, [16]byte{}, tid.Bytes())
	assert.True(t, tid.IsEmpty())

	tid = NewTraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})
	assert.EqualValues(t, [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}, tid.Bytes())
	assert.False(t, tid.IsEmpty())
}

func TestSpanStatusCode(t *testing.T) {
	td := NewTraces()
	rss := td.ResourceSpans()
	rss.Resize(1)
	rss.At(0).InstrumentationLibrarySpans().Resize(1)
	rss.At(0).InstrumentationLibrarySpans().At(0).Spans().Resize(1)
	status := rss.At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).Status()

	// Check handling of deprecated status code, see spec here:
	// https://github.com/open-telemetry/opentelemetry-proto/blob/59c488bfb8fb6d0458ad6425758b70259ff4a2bd/opentelemetry/proto/trace/v1/trace.proto#L231
	//
	// 2. New senders, which are aware of the `code` field MUST set both the
	// `deprecated_code` and `code` fields according to the following rules:
	//
	//   if code==STATUS_CODE_UNSET then `deprecated_code` MUST be
	//   set to DEPRECATED_STATUS_CODE_OK.
	status.orig.DeprecatedCode = otlptrace.Status_DEPRECATED_STATUS_CODE_UNKNOWN_ERROR
	status.SetCode(StatusCodeUnset)
	assert.EqualValues(t, otlptrace.Status_DEPRECATED_STATUS_CODE_OK, status.orig.DeprecatedCode)

	//   if code==STATUS_CODE_OK then `deprecated_code` MUST be
	//   set to DEPRECATED_STATUS_CODE_OK.
	status.orig.DeprecatedCode = otlptrace.Status_DEPRECATED_STATUS_CODE_UNKNOWN_ERROR
	status.SetCode(StatusCodeOk)
	assert.EqualValues(t, otlptrace.Status_DEPRECATED_STATUS_CODE_OK, status.orig.DeprecatedCode)

	//   if code==STATUS_CODE_ERROR then `deprecated_code` MUST be
	//   set to DEPRECATED_STATUS_CODE_UNKNOWN_ERROR.
	status.orig.DeprecatedCode = otlptrace.Status_DEPRECATED_STATUS_CODE_OK
	status.SetCode(StatusCodeError)
	assert.EqualValues(t, otlptrace.Status_DEPRECATED_STATUS_CODE_UNKNOWN_ERROR, status.orig.DeprecatedCode)
}

func TestToFromOtlp(t *testing.T) {
	otlp := []*otlptrace.ResourceSpans(nil)
	td := TracesFromOtlp(otlp)
	assert.EqualValues(t, NewTraces(), td)
	assert.EqualValues(t, otlp, TracesToOtlp(td))
	// More tests in ./tracedata/trace_test.go. Cannot have them here because of
	// circular dependency.
}

func TestResourceSpansWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate ResourceSpans as pdata struct.
	pdataRS := generateTestResourceSpans()

	// Marshal its underlying ProtoBuf to wire.
	wire1, err := gogoproto.Marshal(pdataRS.orig)
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
	var gogoprotoRS2 otlptrace.ResourceSpans
	err = gogoproto.Unmarshal(wire2, &gogoprotoRS2)
	assert.NoError(t, err)

	// Now compare that the original and final ProtoBuf messages are the same.
	// This proves that goproto and gogoproto marshaling/unmarshaling are wire compatible.
	assert.EqualValues(t, pdataRS.orig, &gogoprotoRS2)
}

func TestTracesToFromOtlpProtoBytes(t *testing.T) {
	send := NewTraces()
	fillTestResourceSpansSlice(send.ResourceSpans())
	bytes, err := send.ToOtlpProtoBytes()
	assert.NoError(t, err)

	recv := NewTraces()
	err = recv.FromOtlpProtoBytes(bytes)
	assert.NoError(t, err)
	assert.EqualValues(t, send, recv)
}

func TestTracesFromInvalidOtlpProtoBytes(t *testing.T) {
	err := NewTraces().FromOtlpProtoBytes([]byte{0xFF})
	assert.EqualError(t, err, "unexpected EOF")
}

func TestTracesClone(t *testing.T) {
	traces := NewTraces()
	fillTestResourceSpansSlice(traces.ResourceSpans())
	assert.EqualValues(t, traces, traces.Clone())
}

func BenchmarkTracesClone(b *testing.B) {
	traces := NewTraces()
	fillTestResourceSpansSlice(traces.ResourceSpans())
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		clone := traces.Clone()
		if clone.ResourceSpans().Len() != traces.ResourceSpans().Len() {
			b.Fail()
		}
	}
}

func BenchmarkTracesToOtlp(b *testing.B) {
	traces := NewTraces()
	fillTestResourceSpansSlice(traces.ResourceSpans())
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		buf, err := traces.ToOtlpProtoBytes()
		require.NoError(b, err)
		assert.NotEqual(b, 0, len(buf))
	}
}

func BenchmarkTracesFromOtlp(b *testing.B) {
	baseTraces := NewTraces()
	fillTestResourceSpansSlice(baseTraces.ResourceSpans())
	buf, err := baseTraces.ToOtlpProtoBytes()
	require.NoError(b, err)
	assert.NotEqual(b, 0, len(buf))
	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		traces := NewTraces()
		require.NoError(b, traces.FromOtlpProtoBytes(buf))
		assert.Equal(b, baseTraces.ResourceSpans().Len(), traces.ResourceSpans().Len())
	}
}
