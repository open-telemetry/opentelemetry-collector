// Copyright The OpenTelemetry Authors
// Copyright 2014 gRPC authors.
// SPDX-License-Identifier: Apache-2.0

package configgrpc

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/config/configgrpc/internal/grpccompression/snappy"
	"go.opentelemetry.io/collector/config/configgrpc/internal/grpccompression/zstd"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func setPersistGzipFeatureGateIfRegistered(b *testing.B, enabled bool) {
	b.Helper()
	const gateID = "pkg.configgrpc.PersistGzipCompressors"
	var prev, found bool
	featuregate.GlobalRegistry().VisitAll(func(g *featuregate.Gate) {
		if g.ID() == gateID {
			found = true
			prev = g.IsEnabled()
		}
	})
	if found {
		require.NoError(b, featuregate.GlobalRegistry().Set(gateID, enabled))
		b.Cleanup(func() {
			require.NoError(b, featuregate.GlobalRegistry().Set(gateID, prev))
		})
	}
}

func resetRetainedWritersForBenchmark(compressor encoding.Compressor) {
	if cleaner, ok := any(compressor).(interface{ clearRetainedWriters() }); ok {
		cleaner.clearRetainedWriters()
		runtime.GC()
	}
}

func BenchmarkGzipCompressor(b *testing.B) {
	marshaler := &plog.ProtoMarshaler{}
	payload, err := marshaler.MarshalLogs(testdata.GenerateLogs(50))
	require.NoError(b, err)

	compressor := encoding.GetCompressor(gzip.Name)
	require.NotNil(b, compressor)

	// Single-goroutine tight loop with no idle gaps and no forced GC.
	// Measures baseline single-threaded checkout/reset/checkin overhead and L1/L2
	// cache reuse when the same writer is popped and returned back-to-back.
	b.Run("steady_state", func(b *testing.B) {
		for _, persist := range []bool{false, true} {
			b.Run(fmt.Sprintf("persist_gzip=%t", persist), func(b *testing.B) {
				setPersistGzipFeatureGateIfRegistered(b, persist)
				warm, err := compressor.Compress(io.Discard)
				require.NoError(b, err)
				_, err = warm.Write(payload)
				require.NoError(b, err)
				require.NoError(b, warm.Close())

				b.ReportAllocs()
				b.SetBytes(int64(len(payload)))
				b.ResetTimer()
				for b.Loop() {
					z, err := compressor.Compress(io.Discard)
					if err != nil {
						b.Fatal(err)
					}
					if _, err := z.Write(payload); err != nil {
						b.Fatal(err)
					}
					if err := z.Close(); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
		resetRetainedWritersForBenchmark(compressor)
	})

	// GOMAXPROCS goroutines spinning continuously with no idle gaps and no forced GC.
	// Unlike concurrent_workers_across_gc, workers never park between batches so
	// sync.Pool entries are not evicted by GC; this isolates mutex contention on
	// the shared writer stack against lock-free per-P sync.Pool hits.
	b.Run("parallel", func(b *testing.B) {
		for _, persist := range []bool{false, true} {
			b.Run(fmt.Sprintf("persist_gzip=%t", persist), func(b *testing.B) {
				setPersistGzipFeatureGateIfRegistered(b, persist)
				b.ReportAllocs()
				b.SetBytes(int64(len(payload)))
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						z, err := compressor.Compress(io.Discard)
						if err != nil {
							b.Fatal(err)
						}
						if _, err := z.Write(payload); err != nil {
							b.Fatal(err)
						}
						if err := z.Close(); err != nil {
							b.Fatal(err)
						}
					}
				})
			})
		}
		resetRetainedWritersForBenchmark(compressor)
	})

	// Fixed pool of persistent export workers (within the GOMAXPROCS retention cap)
	// that park idle between batches while periodic two-cycle GCs run.
	// Unlike parallel, idle gaps across two GC cycles sweep sync.Pool's primary and
	// victim caches (modeling receiver/processor allocation churn during batch flush
	// windows and RPC round-trips), testing writer retention across GC cycles.
	b.Run("concurrent_workers_across_gc", func(b *testing.B) {
		for _, persist := range []bool{false, true} {
			b.Run(fmt.Sprintf("persist_gzip=%t", persist), func(b *testing.B) {
				setPersistGzipFeatureGateIfRegistered(b, persist)
				const workers = 8
				startCh := make(chan struct{})
				doneCh := make(chan struct{}, workers)
				var wg sync.WaitGroup
				wg.Add(workers)
				for range workers {
					go func() {
						defer wg.Done()
						for range startCh {
							z, err := compressor.Compress(io.Discard)
							if err != nil {
								b.Error(err)
								doneCh <- struct{}{}
								continue
							}
							if _, err := z.Write(payload); err != nil {
								b.Error(err)
								doneCh <- struct{}{}
								continue
							}
							if err := z.Close(); err != nil {
								b.Error(err)
							}
							doneCh <- struct{}{}
						}
					}()
				}
				b.Cleanup(func() {
					close(startCh)
					wg.Wait()
				})

				b.ReportAllocs()
				b.SetBytes(int64(len(payload)) * workers)
				b.ResetTimer()
				for i := 0; b.Loop(); i++ {
					if i%4 == 3 {
						runtime.GC()
						runtime.GC()
					}
					for range workers {
						startCh <- struct{}{}
					}
					for range workers {
						<-doneCh
					}
				}
			})
		}
		resetRetainedWritersForBenchmark(compressor)
	})

	// Lockstep burst of 2*GOMAXPROCS concurrent workers holding writers open
	// simultaneously across periodic two-cycle GCs.
	// Unlike concurrent_workers_across_gc (which stays within the GOMAXPROCS cap),
	// half of the active writers exceed maxRetained and spill into overflowPool
	// (sync.Pool), testing reuse between GCs and partial eviction across GCs.
	b.Run("overflow_workers_across_gc", func(b *testing.B) {
		for _, persist := range []bool{false, true} {
			b.Run(fmt.Sprintf("persist_gzip=%t", persist), func(b *testing.B) {
				setPersistGzipFeatureGateIfRegistered(b, persist)
				workers := 2 * max(runtime.GOMAXPROCS(0), 1)
				startCh := make(chan struct{}, workers)
				acquiredCh := make(chan struct{}, workers)
				releaseCh := make(chan struct{}, workers)
				doneCh := make(chan struct{}, workers)
				var wg sync.WaitGroup
				wg.Add(workers)
				for range workers {
					go func() {
						defer wg.Done()
						for range startCh {
							z, err := compressor.Compress(io.Discard)
							if err != nil {
								b.Error(err)
							}
							if _, err := z.Write(payload); err != nil {
								b.Error(err)
							}
							acquiredCh <- struct{}{}
							<-releaseCh
							if err := z.Close(); err != nil {
								b.Error(err)
							}
							doneCh <- struct{}{}
						}
					}()
				}
				b.Cleanup(func() {
					close(startCh)
					wg.Wait()
				})

				b.ReportAllocs()
				b.SetBytes(int64(len(payload)) * int64(workers))
				b.ResetTimer()
				for i := 0; b.Loop(); i++ {
					if i%4 == 3 {
						runtime.GC()
						runtime.GC()
					}
					for range workers {
						startCh <- struct{}{}
					}
					for range workers {
						<-acquiredCh
					}
					for range workers {
						releaseCh <- struct{}{}
					}
					for range workers {
						<-doneCh
					}
				}
			})
		}
		resetRetainedWritersForBenchmark(compressor)
	})
}

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
				if compressor.Name() == gzip.Name {
					for _, persist := range []bool{false, true} {
						b.Run(fmt.Sprintf("persist_gzip=%t", persist), func(b *testing.B) {
							setPersistGzipFeatureGateIfRegistered(b, persist)
							_, err := compress(compressor, messageBytes)
							require.NoError(b, err, "compress(_) returned an error")
							b.ReportAllocs()
							b.ResetTimer()
							for b.Loop() {
								require.NoError(b, err, "marshal(_) returned an error")
								_, err := compress(compressor, messageBytes)
								require.NoError(b, err, "compress(_) returned an error")
							}
						})
					}
					resetRetainedWritersForBenchmark(compressor)
					return
				}
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
