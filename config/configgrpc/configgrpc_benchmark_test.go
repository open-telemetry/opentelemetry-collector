// Copyright The OpenTelemetry Authors
// Copyright 2014 gRPC authors.
// SPDX-License-Identifier: Apache-2.0

package configgrpc

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/mostynb/go-grpc-compression/nonclobbering/snappy"
	"github.com/mostynb/go-grpc-compression/nonclobbering/zstd"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func BenchmarkCompressors(b *testing.B) {
	payloads := setupTestPayloads()

	compressors := make([]encoding.Compressor, 0)
	compressors = append(compressors,
		encoding.GetCompressor(gzip.Name),
		encoding.GetCompressor(zstd.Name),
		encoding.GetCompressor(snappy.Name))

	for _, payload := range payloads {
		for _, compressor := range compressors {
			fmt.Println(payload.name)
			messageBytes, err := payload.marshaler.marshal(payload.message)
			require.NoError(b, err, "marshal(_) returned an error")

			compressedBytes, err := compress(compressor, messageBytes)
			require.NoError(b, err, "Compressor.Compress(_) returned an error")

			name := fmt.Sprintf("%v/raw_bytes_%v/compressed_bytes_%v/compressor_%v", payload.name, len(messageBytes), len(compressedBytes), compressor.Name())

			b.Run(name, func(b *testing.B) {
				b.ResetTimer()
				for b.Loop() {
					require.NoError(b, err, "marshal(_) returned an error")
					_, err := compress(compressor, messageBytes)
					require.NoError(b, err, "compress(_) returned an error")
				}
			})
		}
	}
}

func compress(compressor encoding.Compressor, in []byte) ([]byte, error) {
	if compressor == nil {
		return nil, nil
	}
	wrapErr := func(err error) error {
		return status.Errorf(codes.Internal, "error while compressing: %v", err.Error())
	}
	cbuf := &bytes.Buffer{}
	z, err := compressor.Compress(cbuf)
	if err != nil {
		return nil, wrapErr(err)
	}
	if _, err := z.Write(in); err != nil {
		return nil, wrapErr(err)
	}
	if err := z.Close(); err != nil {
		return nil, wrapErr(err)
	}
	return cbuf.Bytes(), nil
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

	// log payloads
	logMarshaler := &logMarshaler{Marshaler: &plog.ProtoMarshaler{}}
	payloads = append(payloads,
		testPayload{
			name:      "sm_log_request",
			message:   testdata.GenerateLogs(1),
			marshaler: logMarshaler,
		},
		testPayload{
			name:      "md_log_request",
			message:   testdata.GenerateLogs(2),
			marshaler: logMarshaler,
		},
		testPayload{
			name:      "lg_log_request",
			message:   testdata.GenerateLogs(50),
			marshaler: logMarshaler,
		})

	// trace payloads
	tracesMarshaler := &traceMarshaler{Marshaler: &ptrace.ProtoMarshaler{}}
	payloads = append(payloads,
		testPayload{
			name:      "sm_trace_request",
			message:   testdata.GenerateTraces(1),
			marshaler: tracesMarshaler,
		},
		testPayload{
			name:      "md_trace_request",
			message:   testdata.GenerateTraces(2),
			marshaler: tracesMarshaler,
		},
		testPayload{
			name:      "lg_trace_request",
			message:   testdata.GenerateTraces(50),
			marshaler: tracesMarshaler,
		})

	// metric payloads
	metricsMarshaler := &metricsMarshaler{Marshaler: &pmetric.ProtoMarshaler{}}
	payloads = append(payloads,
		testPayload{
			name:      "sm_metric_request",
			message:   testdata.GenerateMetrics(1),
			marshaler: metricsMarshaler,
		},
		testPayload{
			name:      "md_metric_request",
			message:   testdata.GenerateMetrics(2),
			marshaler: metricsMarshaler,
		},
		testPayload{
			name:      "lg_metric_request",
			message:   testdata.GenerateMetrics(50),
			marshaler: metricsMarshaler,
		})

	return payloads
}
