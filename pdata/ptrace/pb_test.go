// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptrace

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gootlptrace "go.opentelemetry.io/proto/slim/otlp/trace/v1"
	goproto "google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestTracesProtoWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate Traces as pdata struct.
	td := generateTestTraces()

	// Marshal its underlying ProtoBuf to wire.
	marshaler := &ProtoMarshaler{}
	wire1, err := marshaler.MarshalTraces(td)
	require.NoError(t, err)
	assert.NotNil(t, wire1)

	// Unmarshal from the wire to OTLP Protobuf in goproto's representation.
	var goprotoMessage gootlptrace.TracesData
	err = goproto.Unmarshal(wire1, &goprotoMessage)
	require.NoError(t, err)

	// Marshal to the wire again.
	wire2, err := goproto.Marshal(&goprotoMessage)
	require.NoError(t, err)
	assert.NotNil(t, wire2)

	// Unmarshal from the wire into gogoproto's representation.
	var td2 Traces
	unmarshaler := &ProtoUnmarshaler{}
	td2, err = unmarshaler.UnmarshalTraces(wire2)
	require.NoError(t, err)

	// Now compare that the original and final ProtoBuf messages are the same.
	// This proves that goproto and gogoproto marshaling/unmarshaling are wire compatible.
	assert.Equal(t, td, td2)
}

func TestProtoTracesUnmarshalerError(t *testing.T) {
	p := &ProtoUnmarshaler{}
	_, err := p.UnmarshalTraces([]byte("+$%"))
	assert.Error(t, err)
}

func TestProtoSizer(t *testing.T) {
	marshaler := &ProtoMarshaler{}
	td := NewTraces()
	rms := td.ResourceSpans()
	rms.AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("foo")

	size := marshaler.TracesSize(td)

	bytes, err := marshaler.MarshalTraces(td)
	require.NoError(t, err)
	assert.Equal(t, len(bytes), size)
}

func TestProtoSizerEmptyTraces(t *testing.T) {
	sizer := &ProtoMarshaler{}
	assert.Equal(t, 0, sizer.TracesSize(NewTraces()))
}

func BenchmarkTracesToProto2k(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	traces := generateBenchmarkTraces(2_000)

	for b.Loop() {
		buf, err := marshaler.MarshalTraces(traces)
		require.NoError(b, err)
		assert.NotEmpty(b, buf)
	}
}

func BenchmarkTracesFromProto2k(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	unmarshaler := &ProtoUnmarshaler{}
	baseTraces := generateBenchmarkTraces(2_000)
	buf, err := marshaler.MarshalTraces(baseTraces)
	require.NoError(b, err)
	assert.NotEmpty(b, buf)

	b.ReportAllocs()
	for b.Loop() {
		traces, err := unmarshaler.UnmarshalTraces(buf)
		require.NoError(b, err)
		assert.Equal(b, baseTraces.ResourceSpans().Len(), traces.ResourceSpans().Len())
	}
}

func generateBenchmarkTraces(metricsCount int) Traces {
	now := time.Now()
	startTime := pcommon.NewTimestampFromTime(now.Add(-10 * time.Second))
	endTime := pcommon.NewTimestampFromTime(now)

	md := NewTraces()
	ilm := md.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty()
	ilm.Spans().EnsureCapacity(metricsCount)
	for range metricsCount {
		im := ilm.Spans().AppendEmpty()
		im.SetName("test_name")
		im.SetStartTimestamp(startTime)
		im.SetEndTimestamp(endTime)
	}
	return md
}
