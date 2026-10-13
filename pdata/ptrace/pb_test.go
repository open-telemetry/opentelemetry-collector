// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptrace

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gootlptrace "go.opentelemetry.io/proto/slim/otlp/trace/v1"
	goproto "google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/internal/metadata"
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

func BenchmarkTracesFromProto10MB(b *testing.B) {
	benchmarkFromProto10MB(b, func(n int) []byte {
		td := generateBenchmarkTracesPayload(n)
		buf, err := (&ProtoMarshaler{}).MarshalTraces(td)
		require.NoError(b, err)
		td.getState().DropArena()
		return buf
	}, func(buf []byte) func() {
		td, err := (&ProtoUnmarshaler{}).UnmarshalTraces(buf)
		require.NoError(b, err)
		return func() { td.getState().DropArena() }
	})
}

const protoSize10MB = 10 << 20

func benchmarkFromProto10MB(b *testing.B, gen func(n int) []byte, unmarshalNew func([]byte) func()) {
	for _, pooling := range []bool{false, true} {
		b.Run(fmt.Sprintf("pooling=%v", pooling), func(b *testing.B) {
			prev := metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
			require.NoError(b, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), pooling))
			b.Cleanup(func() {
				require.NoError(b, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), prev))
			})

			buf := protoBufAtLeast(b, protoSize10MB, gen)
			var mBefore, mAfter runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&mBefore)
			release := unmarshalNew(buf)
			runtime.ReadMemStats(&mAfter)
			logHeapDelta(b, buf, pooling, mBefore, mAfter)
			release()

			b.SetBytes(int64(len(buf)))
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				unmarshalNew(buf)()
			}
		})
	}
}

func logHeapDelta(b *testing.B, buf []byte, pooling bool, mBefore, mAfter runtime.MemStats) {
	heapDelta := uint64(0)
	if mAfter.HeapAlloc > mBefore.HeapAlloc {
		heapDelta = mAfter.HeapAlloc - mBefore.HeapAlloc
	}
	b.Logf("wire_bytes=%d pooling=%v heapdeltaB=%d totalallocB=%d", len(buf), pooling, heapDelta, mAfter.TotalAlloc-mBefore.TotalAlloc)
}

func protoBufAtLeast(b *testing.B, target int, gen func(n int) []byte) []byte {
	n := 2_000
	buf := gen(n)
	require.NotEmpty(b, buf)
	n = int(float64(n)*float64(target)/float64(len(buf))) + 1
	buf = gen(n)
	for len(buf) < target {
		n = n*target/len(buf) + n/10 + 1
		buf = gen(n)
	}
	return buf
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

func generateBenchmarkTracesPayload(n int) Traces {
	now := time.Now()
	startTime := pcommon.NewTimestampFromTime(now.Add(-10 * time.Second))
	endTime := pcommon.NewTimestampFromTime(now)

	td := NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", "bench-service")
	ss := rs.ScopeSpans().AppendEmpty()
	ss.Spans().EnsureCapacity(n)
	for range n {
		sp := ss.Spans().AppendEmpty()
		sp.SetName("benchmark-operation-with-a-reasonably-long-name")
		sp.SetStartTimestamp(startTime)
		sp.SetEndTimestamp(endTime)
		sp.Attributes().PutStr("http.route", "/api/v1/resource/{id}")
		sp.Attributes().PutStr("peer.service", "downstream")
	}
	return td
}
