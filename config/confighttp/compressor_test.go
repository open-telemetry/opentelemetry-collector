// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// This file contains helper functions regarding compression/decompression for confighttp.

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/internal/testutil"
)

func BenchmarkCompression(b *testing.B) {
	testutil.SkipGCHeavyBench(b)
	benchmarks := []struct {
		codec    configcompression.Type
		name     string
		function func(*testing.B, configcompression.Type, *bytes.Buffer, []byte)
	}{
		{
			codec:    configcompression.TypeZstd,
			name:     "zstdWithConcurrency",
			function: benchmarkCompression,
		},
		{
			codec:    configcompression.TypeZstd,
			name:     "zstdNoConcurrency",
			function: benchmarkCompressionNoConcurrency,
		},
	}
	payload := make([]byte, 10<<22)
	buffer := bytes.Buffer{}
	buffer.Grow(len(payload))

	ts := &bytes.Buffer{}
	defer func() {
		fmt.Printf("input => %.2f MB\n", float64(len(payload))/(1024*1024))
		fmt.Println(ts)
	}()

	for i := range benchmarks {
		benchmark := &benchmarks[i]
		b.Run(benchmark.name, func(b *testing.B) {
			benchmark.function(b, benchmark.codec, &buffer, payload)
		})
	}
}

func benchmarkCompression(b *testing.B, _ configcompression.Type, buf *bytes.Buffer, payload []byte) {
	// Concurrency Enabled
	stringReader := strings.NewReader(string(payload))
	stringReadCloser := io.NopCloser(stringReader)
	var enc io.Writer
	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	for b.Loop() {
		enc, _ = zstd.NewWriter(nil, zstd.WithEncoderConcurrency(5))
		enc.(writeCloserReset).Reset(buf)
		_, copyErr := io.Copy(enc, stringReadCloser)
		require.NoError(b, copyErr)
	}
}

func benchmarkCompressionNoConcurrency(b *testing.B, _ configcompression.Type, buf *bytes.Buffer, payload []byte) {
	stringReader := strings.NewReader(string(payload))
	stringReadCloser := io.NopCloser(stringReader)
	var enc io.Writer
	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	for b.Loop() {
		enc, _ = zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1))
		enc.(writeCloserReset).Reset(buf)
		_, copyErr := io.Copy(enc, stringReadCloser)
		require.NoError(b, copyErr)
	}
}
