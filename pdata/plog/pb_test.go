// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plog

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gootlplogs "go.opentelemetry.io/proto/slim/otlp/logs/v1"
	goproto "google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/internal/metadata"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestLogsProtoWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate Logs as pdata struct.
	td := generateTestLogs()

	// Marshal its underlying ProtoBuf to wire.
	marshaler := &ProtoMarshaler{}
	wire1, err := marshaler.MarshalLogs(td)
	require.NoError(t, err)
	assert.NotNil(t, wire1)

	// Unmarshal from the wire to OTLP Protobuf in goproto's representation.
	var goprotoMessage gootlplogs.LogsData
	err = goproto.Unmarshal(wire1, &goprotoMessage)
	require.NoError(t, err)

	// Marshal to the wire again.
	wire2, err := goproto.Marshal(&goprotoMessage)
	require.NoError(t, err)
	assert.NotNil(t, wire2)

	// Unmarshal from the wire into gogoproto's representation.
	var td2 Logs
	unmarshaler := &ProtoUnmarshaler{}
	td2, err = unmarshaler.UnmarshalLogs(wire2)
	require.NoError(t, err)

	// Now compare that the original and final ProtoBuf messages are the same.
	// This proves that goproto and gogoproto marshaling/unmarshaling are wire compatible.
	assert.Equal(t, td, td2)
}

func TestProtoLogsUnmarshalerError(t *testing.T) {
	p := &ProtoUnmarshaler{}
	_, err := p.UnmarshalLogs([]byte("+$%"))
	assert.Error(t, err)
}

func TestProtoSizer(t *testing.T) {
	marshaler := &ProtoMarshaler{}
	ld := NewLogs()
	ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().SetSeverityText("error")

	size := marshaler.LogsSize(ld)

	bytes, err := marshaler.MarshalLogs(ld)
	require.NoError(t, err)
	assert.Equal(t, len(bytes), size)
}

func TestProtoSizerEmptyLogs(t *testing.T) {
	sizer := &ProtoMarshaler{}
	assert.Equal(t, 0, sizer.LogsSize(NewLogs()))
}

func BenchmarkLogsToProto2k(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	logs := generateBenchmarkLogs(2_000)

	for b.Loop() {
		buf, err := marshaler.MarshalLogs(logs)
		require.NoError(b, err)
		assert.NotEmpty(b, buf)
	}
}

func BenchmarkLogsFromProto2k(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	unmarshaler := &ProtoUnmarshaler{}
	baseLogs := generateBenchmarkLogs(2_000)
	buf, err := marshaler.MarshalLogs(baseLogs)
	require.NoError(b, err)
	assert.NotEmpty(b, buf)

	b.ReportAllocs()
	for b.Loop() {
		logs, err := unmarshaler.UnmarshalLogs(buf)
		require.NoError(b, err)
		assert.Equal(b, baseLogs.ResourceLogs().Len(), logs.ResourceLogs().Len())
	}
}

func BenchmarkLogsFromProto10MB(b *testing.B) {
	benchmarkFromProto10MB(b, func(n int) []byte {
		ld := generateBenchmarkLogsPayload(n)
		buf, err := (&ProtoMarshaler{}).MarshalLogs(ld)
		require.NoError(b, err)
		ld.getState().DropArena()
		return buf
	}, func(buf []byte) func() {
		ld, err := (&ProtoUnmarshaler{}).UnmarshalLogs(buf)
		require.NoError(b, err)
		return func() { ld.getState().DropArena() }
	}, func() (func([]byte), func()) {
		ld := NewLogs()
		return func(buf []byte) {
			internal.DeleteExportLogsServiceRequest(ld.getOrig(), false)
			ld.getState().ResetArena()
			ld.getState().RetainWire(buf)
			require.NoError(b, ld.getOrig().UnmarshalProtoState(buf, ld.getState()))
		}, func() { ld.getState().DropArena() }
	})
}

const protoSize10MB = 10 << 20

func benchmarkFromProto10MB(b *testing.B, gen func(n int) []byte, unmarshalNew func([]byte) func(), newReuse func() (into func([]byte), release func())) {
	for _, pooling := range []bool{false, true} {
		for _, reuse := range []bool{false, true} {
			b.Run(fmt.Sprintf("pooling=%v/reuse=%v", pooling, reuse), func(b *testing.B) {
				prev := metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
				require.NoError(b, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), pooling))
				b.Cleanup(func() {
					require.NoError(b, featuregate.GlobalRegistry().Set(metadata.PdataUseProtoPoolingFeatureGate.ID(), prev))
				})

				buf := protoBufAtLeast(b, protoSize10MB, gen)
				var mBefore, mAfter runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&mBefore)
				if reuse {
					into, release := newReuse()
					into(buf)
					runtime.ReadMemStats(&mAfter)
					logHeapDelta(b, buf, pooling, reuse, mBefore, mAfter)
					b.SetBytes(int64(len(buf)))
					b.ReportAllocs()
					b.ResetTimer()
					for b.Loop() {
						into(buf)
					}
					b.StopTimer()
					release()
					return
				}
				release := unmarshalNew(buf)
				runtime.ReadMemStats(&mAfter)
				logHeapDelta(b, buf, pooling, reuse, mBefore, mAfter)
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
}

func logHeapDelta(b *testing.B, buf []byte, pooling, reuse bool, mBefore, mAfter runtime.MemStats) {
	heapDelta := uint64(0)
	if mAfter.HeapAlloc > mBefore.HeapAlloc {
		heapDelta = mAfter.HeapAlloc - mBefore.HeapAlloc
	}
	b.Logf("wire_bytes=%d pooling=%v reuse=%v heapdeltaB=%d totalallocB=%d", len(buf), pooling, reuse, heapDelta, mAfter.TotalAlloc-mBefore.TotalAlloc)
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

func generateBenchmarkLogs(logsCount int) Logs {
	endTime := pcommon.NewTimestampFromTime(time.Now())

	md := NewLogs()
	ilm := md.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty()
	ilm.LogRecords().EnsureCapacity(logsCount)
	for range logsCount {
		im := ilm.LogRecords().AppendEmpty()
		im.SetTimestamp(endTime)
	}
	return md
}

func generateBenchmarkLogsPayload(n int) Logs {
	endTime := pcommon.NewTimestampFromTime(time.Now())

	ld := NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	rl.Resource().Attributes().PutStr("service.name", "bench-service")
	sl := rl.ScopeLogs().AppendEmpty()
	sl.LogRecords().EnsureCapacity(n)
	for range n {
		lr := sl.LogRecords().AppendEmpty()
		lr.SetTimestamp(endTime)
		lr.SetSeverityText("INFO")
		lr.Body().SetStr("benchmark log body with enough text to exercise string borrow")
		lr.Attributes().PutStr("http.route", "/api/v1/resource/{id}")
		lr.Attributes().PutStr("peer.service", "downstream")
	}
	return ld
}
