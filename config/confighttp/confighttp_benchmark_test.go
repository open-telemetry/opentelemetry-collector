// Copyright The OpenTelemetry Authors
// Copyright 2014 gRPC authors.
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"testing"

	"github.com/golang/snappy"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func BenchmarkCompressors(b *testing.B) {
	payloads := setupTestPayloads()

	compressors := make(map[string]*compressor, 0)
	compressors["gzip"] = gZipPool
	compressors["snappy"] = snappyPool
	compressors["zstd"] = zStdPool
	compressors["lz4"] = lz4Pool

	for _, payload := range payloads {
		for key, compressor := range compressors {
			fmt.Printf(payload.name)
			messageBytes, err := payload.marshaler.marshal(payload.message)
			if err != nil {
				b.Errorf("marshal(_) returned an error")
			}

			buffComp := bytes.NewBuffer([]byte{})
			reader := bytes.NewBuffer(messageBytes)
			readCloser := io.NopCloser(reader)

			err = compressor.compress(buffComp, readCloser)
			if err != nil {
				b.Errorf("Compressor.Compress(_) returned an error")
			}

			err = readCloser.Close()
			if err != nil {
				b.Errorf("readCloser.Close(_) returned an error")
			}

			name := fmt.Sprintf("%v/raw_bytes_%v/compressed_bytes_%v/compressed_rate_%v/compressor_%v", payload.name, len(messageBytes), buffComp.Len(), len(messageBytes)/buffComp.Len(), key)
			b.Run(name, func(b *testing.B) {
				b.ResetTimer()
				buf := bytes.NewBuffer([]byte{})
				reader := bytes.NewReader(messageBytes)
				readCloser := io.NopCloser(reader)
				for i := 0; i < b.N; i++ {
					err = compressor.compress(buf, readCloser)
					if err != nil {
						b.Errorf("compress(_) returned an error")
					}
				}
				err = readCloser.Close()
				if err != nil {
					b.Errorf("Close(_) returned an error")
				}
			})
		}
	}
}

func BenchmarkDecompressors(b *testing.B) {
	const MaxReadSize = 1 << 20
	payloads := setupTestPayloads()

	compressors := make(map[string]*compressor, 0)
	compressors["gzip"] = gZipPool
	compressors["snappy"] = snappyPool
	compressors["zstd"] = zStdPool
	compressors["lz4"] = lz4Pool

	for _, payload := range payloads {
		for key, compressor := range compressors {
			fmt.Printf(payload.name)
			messageBytes, err := payload.marshaler.marshal(payload.message)
			if err != nil {
				b.Errorf("marshal(_) returned an error")
			}

			buffComp := bytes.NewBuffer([]byte{})
			reader := bytes.NewReader(messageBytes)
			readCloser := io.NopCloser(reader)
			err = compressor.compress(buffComp, readCloser)
			if err != nil {
				b.Errorf("Compressor.Compress(_) returned an error")
			}

			err = readCloser.Close()
			if err != nil {
				b.Errorf("readCloser.Close(_) returned an error")
			}

			name := fmt.Sprintf("%vcompressed_rate_%v/compressor_%v", payload.name, len(messageBytes)/buffComp.Len(), key)
			buffDecomp := new(bytes.Buffer)
			// Since the decompression libraries are different, a lot of extra work must be done according to the official decompression method, so each separate decompression is written
			switch key {
			case "gzip":
				b.Run(name, func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						buffDecomp.Reset()
						in := bytes.NewReader(buffComp.Bytes())
						gr, grErr := gzip.NewReader(in)
						if grErr != nil {
							b.Errorf("create(_) returned an error")
						}
						limitedReader := io.LimitReader(gr, MaxReadSize)
						if _, err = io.Copy(buffDecomp, limitedReader); err != nil {
							b.Errorf("copy(_) returned an error")
						}
						assert.True(b, buffDecomp.String() == string(messageBytes), "decompress(_) error")
						err = gr.Close()
						if err != nil {
							b.Errorf("gr.Close(_) returned an error")
						}
					}
				})
			case "snappy":
				b.Run(name, func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						buffDecomp.Reset()
						in := bytes.NewReader(buffComp.Bytes())
						sr := snappy.NewReader(in)
						limitedReader := io.LimitReader(sr, MaxReadSize)
						if _, err = io.Copy(buffDecomp, limitedReader); err != nil {
							b.Errorf("copy(_) returned an error")
						}
						assert.True(b, buffDecomp.String() == string(messageBytes), "decompress(_) error")
					}
				})
			case "zstd":
				b.Run(name, func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						buffDecomp.Reset()
						in := bytes.NewReader(buffComp.Bytes())
						zr, zrErr := zstd.NewReader(
							in,
							// Concurrency 1 disables async decoding. We don't need async decoding, it is pointless
							// for our use-case (a server accepting decoding http requests).
							// Disabling async improves performance (I benchmarked it previously when working
							// on https://github.com/open-telemetry/opentelemetry-collector-contrib/pull/23257).
							zstd.WithDecoderConcurrency(1),
						)
						if zrErr != nil {
							b.Errorf("create(_) returned an error")
						}
						limitedReader := io.LimitReader(zr, MaxReadSize)
						if _, err = io.Copy(buffDecomp, limitedReader); err != nil {
							b.Errorf("copy(_) returned an error")
						}
						assert.True(b, buffDecomp.String() == string(messageBytes), "decompress(_) error")
						zr.Close()
					}
				})
			case "lz4":
				b.Run(name, func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						buffDecomp.Reset()
						in := bytes.NewReader(buffComp.Bytes())
						lr := lz4.NewReader(in)
						limitedReader := io.LimitReader(lr, MaxReadSize)
						if _, err = io.Copy(buffDecomp, limitedReader); err != nil {
							b.Errorf("copy(_) returned an error")
						}
						assert.True(b, buffDecomp.String() == string(messageBytes), "decompress(_) error")
					}
				})
			}
		}
	}
}

type testPayload struct {
	name      string
	message   any
	marshaler marshaler
}

type marshaler interface {
	marshal(any) ([]byte, error)
}

type logMarshaler struct {
	plog.Marshaler
}

func (m *logMarshaler) marshal(e any) ([]byte, error) {
	return m.MarshalLogs(e.(plog.Logs))
}

type traceMarshaler struct {
	ptrace.Marshaler
}

func (m *traceMarshaler) marshal(e any) ([]byte, error) {
	return m.MarshalTraces(e.(ptrace.Traces))
}

type metricsMarshaler struct {
	pmetric.Marshaler
}

func (m *metricsMarshaler) marshal(e any) ([]byte, error) {
	return m.MarshalMetrics(e.(pmetric.Metrics))
}

func setupTestPayloads() []testPayload {
	payloads := make([]testPayload, 0)
	var mx, md, sm = 1000, 100, 10

	// log payloads
	logMarshaler := &logMarshaler{Marshaler: &plog.ProtoMarshaler{}}
	payloads = append(payloads, testPayload{
		name:      "sm_log_request",
		message:   testdata.GenerateLogs(sm),
		marshaler: logMarshaler})
	payloads = append(payloads, testPayload{
		name:      "md_log_request",
		message:   testdata.GenerateLogs(md),
		marshaler: logMarshaler})
	payloads = append(payloads, testPayload{
		name:      "lg_log_request",
		message:   testdata.GenerateLogs(mx),
		marshaler: logMarshaler})

	// trace payloads
	tracesMarshaler := &traceMarshaler{Marshaler: &ptrace.ProtoMarshaler{}}
	payloads = append(payloads, testPayload{
		name:      "sm_trace_request",
		message:   testdata.GenerateTraces(sm),
		marshaler: tracesMarshaler})
	payloads = append(payloads, testPayload{
		name:      "md_trace_request",
		message:   testdata.GenerateTraces(md),
		marshaler: tracesMarshaler})
	payloads = append(payloads, testPayload{
		name:      "lg_trace_request",
		message:   testdata.GenerateTraces(mx),
		marshaler: tracesMarshaler})

	// metric payloads
	metricsMarshaler := &metricsMarshaler{Marshaler: &pmetric.ProtoMarshaler{}}
	payloads = append(payloads, testPayload{
		name:      "sm_metric_request",
		message:   testdata.GenerateMetrics(sm),
		marshaler: metricsMarshaler})
	payloads = append(payloads, testPayload{
		name:      "md_metric_request",
		message:   testdata.GenerateMetrics(md),
		marshaler: metricsMarshaler})
	payloads = append(payloads, testPayload{
		name:      "lg_metric_request",
		message:   testdata.GenerateMetrics(mx),
		marshaler: metricsMarshaler})

	return payloads
}
