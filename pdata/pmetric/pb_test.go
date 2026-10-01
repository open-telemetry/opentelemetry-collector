// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetric

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gootlpmetrics "go.opentelemetry.io/proto/slim/otlp/metrics/v1"
	goproto "google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/internal/metadata"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestMetricsProtoWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate Metrics as pdata struct.
	td := generateTestMetrics()

	// Marshal its underlying ProtoBuf to wire.
	marshaler := &ProtoMarshaler{}
	wire1, err := marshaler.MarshalMetrics(td)
	require.NoError(t, err)
	assert.NotNil(t, wire1)

	// Unmarshal from the wire to OTLP Protobuf in goproto's representation.
	var goprotoMessage gootlpmetrics.MetricsData
	err = goproto.Unmarshal(wire1, &goprotoMessage)
	require.NoError(t, err)

	// Marshal to the wire again.
	wire2, err := goproto.Marshal(&goprotoMessage)
	require.NoError(t, err)
	assert.NotNil(t, wire2)

	// Unmarshal from the wire into gogoproto's representation.
	var td2 Metrics
	unmarshaler := &ProtoUnmarshaler{}
	td2, err = unmarshaler.UnmarshalMetrics(wire2)
	require.NoError(t, err)

	// Now compare that the original and final ProtoBuf messages are the same.
	// This proves that goproto and gogoproto marshaling/unmarshaling are wire compatible.
	assert.Equal(t, td, td2)
}

func TestProtoMetricsUnmarshalerError(t *testing.T) {
	p := &ProtoUnmarshaler{}
	_, err := p.UnmarshalMetrics([]byte("+$%"))
	assert.Error(t, err)
}

func TestProtoSizer(t *testing.T) {
	marshaler := &ProtoMarshaler{}
	md := NewMetrics()
	md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetName("foo")

	size := marshaler.MetricsSize(md)

	bytes, err := marshaler.MarshalMetrics(md)
	require.NoError(t, err)
	assert.Equal(t, len(bytes), size)
}

func TestProtoSizerEmptyMetrics(t *testing.T) {
	sizer := &ProtoMarshaler{}
	assert.Equal(t, 0, sizer.MetricsSize(NewMetrics()))
}

func BenchmarkMetricsToProto2k(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	metrics := generateBenchmarkMetrics(2_000)

	for b.Loop() {
		buf, err := marshaler.MarshalMetrics(metrics)
		require.NoError(b, err)
		assert.NotEmpty(b, buf)
	}
}

func BenchmarkMetricsFromProto10k(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	unmarshaler := &ProtoUnmarshaler{}
	baseMetrics := generateBenchmarkMetrics(2_000)
	buf, err := marshaler.MarshalMetrics(baseMetrics)
	require.NoError(b, err)
	assert.NotEmpty(b, buf)

	b.ReportAllocs()
	for b.Loop() {
		metrics, err := unmarshaler.UnmarshalMetrics(buf)
		require.NoError(b, err)
		assert.Equal(b, baseMetrics.ResourceMetrics().Len(), metrics.ResourceMetrics().Len())
	}
}

func BenchmarkMetricsFromProto10MB(b *testing.B) {
	benchmarkFromProto10MB(b, func(n int) []byte {
		md := generateBenchmarkMetricsPayload(n)
		buf, err := (&ProtoMarshaler{}).MarshalMetrics(md)
		require.NoError(b, err)
		md.getState().DropArena()
		return buf
	}, func(buf []byte) func() {
		md, err := (&ProtoUnmarshaler{}).UnmarshalMetrics(buf)
		require.NoError(b, err)
		return func() { md.getState().DropArena() }
	}, func() (func([]byte), func()) {
		md := NewMetrics()
		return func(buf []byte) {
			internal.DeleteExportMetricsServiceRequest(md.getOrig(), false)
			md.getState().ResetArena()
			md.getState().RetainWire(buf)
			require.NoError(b, md.getOrig().UnmarshalProtoState(buf, md.getState()))
		}, func() { md.getState().DropArena() }
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

func generateBenchmarkMetrics(metricsCount int) Metrics {
	now := time.Now()
	startTime := pcommon.NewTimestampFromTime(now.Add(-10 * time.Second))
	endTime := pcommon.NewTimestampFromTime(now)

	md := NewMetrics()
	ilm := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
	ilm.Metrics().EnsureCapacity(metricsCount)
	for range metricsCount {
		im := ilm.Metrics().AppendEmpty()
		im.SetName("test_name")
		idp := im.SetEmptySum().DataPoints().AppendEmpty()
		idp.SetStartTimestamp(startTime)
		idp.SetTimestamp(endTime)
		idp.SetIntValue(123)
	}
	return md
}

func generateBenchmarkMetricsPayload(n int) Metrics {
	now := time.Now()
	startTime := pcommon.NewTimestampFromTime(now.Add(-10 * time.Second))
	endTime := pcommon.NewTimestampFromTime(now)

	md := NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("service.name", "bench-service")
	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Metrics().EnsureCapacity(n)
	for range n {
		m := sm.Metrics().AppendEmpty()
		m.SetName("benchmark.requests.total")
		m.SetDescription("benchmark metric with a longer description string")
		m.SetUnit("1")
		dp := m.SetEmptySum().DataPoints().AppendEmpty()
		dp.SetStartTimestamp(startTime)
		dp.SetTimestamp(endTime)
		dp.SetIntValue(123)
		dp.Attributes().PutStr("http.route", "/api/v1/resource/{id}")
		dp.Attributes().PutStr("peer.service", "downstream")
	}
	return md
}
