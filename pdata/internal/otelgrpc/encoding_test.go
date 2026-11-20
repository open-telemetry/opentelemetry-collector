// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelgrpc

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/mem"

	"go.opentelemetry.io/collector/pdata/internal"
)

func genMetricsRequestWithSize(minBytes int) *internal.ExportMetricsServiceRequest {
	req := internal.NewExportMetricsServiceRequest()
	for req.SizeProto() < minBytes {
		req.ResourceMetrics = append(req.ResourceMetrics, internal.GenTestResourceMetrics())
	}
	return req
}

func TestCodecV2MarshalRoundTrip(t *testing.T) {
	codec := encoding.GetCodecV2(Name)
	require.NotNil(t, codec)

	req := genMetricsRequestWithSize(48 << 10)
	bufs, err := codec.Marshal(req)
	require.NoError(t, err)
	defer bufs.Free()

	out := internal.NewExportMetricsServiceRequest()
	require.NoError(t, codec.Unmarshal(bufs, out))
	assert.Equal(t, req.SizeProto(), out.SizeProto())
}

// Recommended CLI invocation:
//
//	export bench=marshal-before && go test ./internal/otelgrpc -run '^$' -bench '^BenchmarkCodecMarshal' -benchtime 1s -count 6 -cpu 2 -timeout 999m -benchmem | tee ${bench}.txt
func BenchmarkCodecMarshal(b *testing.B) {
	codec := encoding.GetCodecV2(Name)

	for _, tc := range []struct {
		name     string
		minBytes int
	}{
		{name: "size=48KiB", minBytes: 48 << 10},
		{name: "size=100KiB", minBytes: 100 << 10},
		{name: "size=200KiB", minBytes: 200 << 10},
	} {
		req := genMetricsRequestWithSize(tc.minBytes)
		b.Run(tc.name+"/pool=warm", func(b *testing.B) {
			runtime.GC()
			runtime.GC()
			bufs, err := codec.Marshal(req)
			require.NoError(b, err)
			bufs.Free()

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				bufs, err := codec.Marshal(req)
				if err != nil {
					b.Fatal(err)
				}
				bufs.Free()
			}
		})
		b.Run(tc.name+"/pool=cold", func(b *testing.B) {
			otelBufferPool = mem.NewTieredBufferPool(defaultBufferPoolSizes...)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_, err := codec.Marshal(req)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
