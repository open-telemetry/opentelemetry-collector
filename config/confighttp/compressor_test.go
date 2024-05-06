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

	"go.opentelemetry.io/collector/config/configcompression"
)

func BenchmarkCompression(b *testing.B) {
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
	payload := make([]byte, 10<<20)
	buffer := bytes.Buffer{}
	buffer.Grow(len(payload))

	ts := &bytes.Buffer{}
	defer func() {
		fmt.Printf("input => %.2f MB\n", float64(len(payload))/(1024*1024))
		fmt.Println(ts)
	}()

	for i := range benchmarks {
		benchmark := &benchmarks[i]
		b.Run(fmt.Sprint(benchmark.name), func(b *testing.B) {
			benchmark.function(b, benchmark.codec, &buffer, payload)
		})

	}
}

func benchmarkCompression(b *testing.B, _ configcompression.Type, buf *bytes.Buffer, payload []byte) {
	// Concurrency Enabled

	b.Run("compress", func(b *testing.B) {
		stringReader := strings.NewReader(string(payload))
		stringReadCloser := io.NopCloser(stringReader)
		var enc io.Writer
		b.ResetTimer()
		b.ReportAllocs()
		b.SetBytes(int64(len(payload)))
		for i := 0; i < b.N; i++ {
			enc, _ = zstd.NewWriter(nil, zstd.WithEncoderConcurrency(5))
			enc.(writeCloserReset).Reset(buf)
			_, copyErr := io.Copy(enc, stringReadCloser)
			if copyErr != nil {
				b.Fatal(copyErr)
			}
		}
	})
}

func benchmarkCompressionNoConcurrency(b *testing.B, _ configcompression.Type, buf *bytes.Buffer, payload []byte) {
	// Concurrency Disabled

	b.Run("compress", func(b *testing.B) {
		stringReader := strings.NewReader(string(payload))
		stringReadCloser := io.NopCloser(stringReader)
		var enc io.Writer
		b.ResetTimer()
		b.ReportAllocs()
		b.SetBytes(int64(len(payload)))
		for i := 0; i < b.N; i++ {
			enc, _ = zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1))
			enc.(writeCloserReset).Reset(buf)
			_, copyErr := io.Copy(enc, stringReadCloser)
			if copyErr != nil {
				b.Fatal(copyErr)
			}
		}
	})
}
