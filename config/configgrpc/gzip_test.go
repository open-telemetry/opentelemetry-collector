// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configgrpc

import (
	"bytes"
	"io"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/encoding"
	grpcgzip "google.golang.org/grpc/encoding/gzip"

	"go.opentelemetry.io/collector/config/configgrpc/internal/metadata"
	"go.opentelemetry.io/collector/featuregate"
)

func setPersistGzipFeatureGateForTest(tb testing.TB, enabled bool) {
	tb.Helper()
	gate := metadata.PkgConfiggrpcPersistGzipCompressorsFeatureGate
	prev := gate.IsEnabled()
	require.NoError(tb, featuregate.GlobalRegistry().Set(gate.ID(), enabled))
	tb.Cleanup(func() {
		require.NoError(tb, featuregate.GlobalRegistry().Set(gate.ID(), prev))
	})
}

func (c *gzipCompressor) clearRetainedWriters() {
	c.mu.Lock()
	clear(c.writers)
	c.writers = c.writers[:0]
	c.mu.Unlock()
}

func TestGzipInitRegistration(t *testing.T) {
	comp := encoding.GetCompressor(grpcgzip.Name)
	require.NotNil(t, comp)
	assert.Equal(t, grpcgzip.Name, comp.Name())
	gc, ok := comp.(*gzipCompressor)
	require.True(t, ok, "registered gzip compressor should be *gzipCompressor")
	assert.NotNil(t, gc.fallback, "fallback compressor should be captured from grpc/encoding/gzip")
	assert.False(t, metadata.PkgConfiggrpcPersistGzipCompressorsFeatureGate.IsEnabled(), "alpha feature gate should be disabled by default")
}

func TestGzipFeatureGateDisabledDelegatesToFallback(t *testing.T) {
	setPersistGzipFeatureGateForTest(t, false)

	registered := encoding.GetCompressor(grpcgzip.Name).(*gzipCompressor)
	comp := newGzipCompressor(registered.fallback)

	payload := []byte("fallback delegation payload")
	var buf bytes.Buffer
	w, err := comp.Compress(&buf)
	require.NoError(t, err)
	_, isPersistedWriter := w.(*gzipWriter)
	assert.False(t, isPersistedWriter, "disabled feature gate should return fallback writer")
	_, err = w.Write(payload)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	assert.Empty(t, comp.writers, "disabled feature gate must not retain writers")

	r, err := comp.Decompress(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	_, isPersistedReader := r.(*gzipReader)
	assert.False(t, isPersistedReader, "disabled feature gate should return fallback reader")
	got, err := io.ReadAll(r)
	require.NoError(t, err)
	if closer, ok := r.(io.Closer); ok {
		require.NoError(t, closer.Close())
	}
	assert.Equal(t, payload, got)

	// Verify toggling gate while a writer is checked out returns each writer to its own pool cleanly.
	wFallback, err := comp.Compress(io.Discard)
	require.NoError(t, err)
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PkgConfiggrpcPersistGzipCompressorsFeatureGate.ID(), true))
	wPersisted, err := comp.Compress(io.Discard)
	require.NoError(t, err)
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.PkgConfiggrpcPersistGzipCompressorsFeatureGate.ID(), false))

	require.NoError(t, wFallback.Close())
	require.NoError(t, wPersisted.Close())
	assert.Len(t, comp.writers, 1)
}

func TestGzipRoundTrip(t *testing.T) {
	setPersistGzipFeatureGateForTest(t, true)
	comp := newGzipCompressor(nil)
	for _, tt := range []struct {
		name    string
		payload []byte
	}{
		{name: "empty", payload: nil},
		{name: "small", payload: []byte("message request gzip")},
		{name: "large", payload: bytes.Repeat([]byte("collector otlp gzip payload "), 4096)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w, err := comp.Compress(&buf)
			require.NoError(t, err)
			_, err = w.Write(tt.payload)
			require.NoError(t, err)
			require.NoError(t, w.Close())

			r, err := comp.Decompress(bytes.NewReader(buf.Bytes()))
			require.NoError(t, err)
			closer, ok := r.(io.Closer)
			require.True(t, ok, "decompressor reader must implement io.Closer")

			got, err := io.ReadAll(r)
			require.NoError(t, err)
			require.NoError(t, closer.Close())

			if len(tt.payload) == 0 {
				assert.Empty(t, got)
				return
			}
			assert.Equal(t, tt.payload, got)
		})
	}
}

func TestGzipDecompressErrors(t *testing.T) {
	setPersistGzipFeatureGateForTest(t, true)
	comp := newGzipCompressor(nil)

	// Cold Decompress error when readerPool is empty (gzip.NewReader fails).
	_, err := comp.Decompress(bytes.NewReader([]byte("not-gzip")))
	require.Error(t, err)

	// Populate readerPool with one valid reader.
	var valid bytes.Buffer
	w, err := comp.Compress(&valid)
	require.NoError(t, err)
	_, err = w.Write([]byte("seed"))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	r, err := comp.Decompress(bytes.NewReader(valid.Bytes()))
	require.NoError(t, err)
	_, err = io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.(io.Closer).Close())

	// Warm Decompress error when z.Reset fails; reader should be returned to pool.
	_, err = comp.Decompress(bytes.NewReader([]byte("corrupt-header")))
	require.Error(t, err)

	// Subsequent valid Decompress still succeeds.
	r2, err := comp.Decompress(bytes.NewReader(valid.Bytes()))
	require.NoError(t, err)
	got, err := io.ReadAll(r2)
	require.NoError(t, err)
	require.NoError(t, r2.(io.Closer).Close())
	assert.Equal(t, []byte("seed"), got)
}

func TestGzipWriterLIFOAndRetentionAcrossGC(t *testing.T) {
	setPersistGzipFeatureGateForTest(t, true)
	comp := &gzipCompressor{
		writers:     make([]*gzipWriter, 0, 2),
		maxRetained: 2,
	}

	// Checkout 3 writers concurrently (exceeding maxRetained=2 by 1).
	w1, err := comp.Compress(io.Discard)
	require.NoError(t, err)
	w2, err := comp.Compress(io.Discard)
	require.NoError(t, err)
	w3, err := comp.Compress(io.Discard)
	require.NoError(t, err)

	// Return in order w1, w2, w3: w1 and w2 fill c.writers (with w2 at LIFO top),
	// while w3 spills to overflowPool.
	require.NoError(t, w1.Close())
	require.NoError(t, w2.Close())
	require.NoError(t, w3.Close())
	assert.Len(t, comp.writers, 2)

	// Force two GC cycles to evict sync.Pool (overflowPool); c.writers must survive.
	runtime.GC()
	runtime.GC()

	gotFirst, err := comp.Compress(io.Discard)
	require.NoError(t, err)
	assert.Same(t, w2, gotFirst, "Compress must pop the most recently returned writer (LIFO) and retain across GC")

	gotSecond, err := comp.Compress(io.Discard)
	require.NoError(t, err)
	assert.Same(t, w1, gotSecond, "second Compress must pop the next retained writer")

	require.NoError(t, gotSecond.Close())
	require.NoError(t, gotFirst.Close())
}

func TestGzipOverflowPoolReuse(t *testing.T) {
	setPersistGzipFeatureGateForTest(t, true)

	t.Run("spillover_and_reuse", func(t *testing.T) {
		comp := &gzipCompressor{
			writers:     make([]*gzipWriter, 0, 2),
			maxRetained: 2,
		}
		const total = 18
		open := make([]io.WriteCloser, total)
		for i := range total {
			w, err := comp.Compress(io.Discard)
			require.NoError(t, err)
			open[i] = w
		}
		// Closing all 18 retains 2 in comp.writers and spills 16 into overflowPool.
		for _, w := range open {
			require.NoError(t, w.Close())
		}
		assert.Len(t, comp.writers, 2)

		// Checking out 18 again drains comp.writers (2) and pops from overflowPool.
		for i := range total {
			w, err := comp.Compress(io.Discard)
			require.NoError(t, err)
			open[i] = w
		}
		for _, w := range open {
			require.NoError(t, w.Close())
		}
	})

	t.Run("deterministic_reset_from_overflow_pool", func(t *testing.T) {
		comp := &gzipCompressor{
			writers:     make([]*gzipWriter, 0, 1),
			maxRetained: 1,
		}
		// Allocate one writer without closing it so comp.writers stays empty.
		seed, err := comp.Compress(io.Discard)
		require.NoError(t, err)
		_, err = seed.Write([]byte("discarded-stream"))
		require.NoError(t, err)
		require.NoError(t, seed.(*gzipWriter).Writer.Close())

		// Guarantee overflowPool.Get() returns non-nil even under -race (which randomly
		// drops sync.Pool.Put items) or if a background GC runs mid-test.
		comp.overflowPool.New = func() any { return seed }
		t.Cleanup(func() { comp.overflowPool.New = nil })

		payload := []byte("overflow-pool-reset-check")
		var buf bytes.Buffer
		got, err := comp.Compress(&buf)
		require.NoError(t, err)
		assert.Same(t, seed, got, "Compress should reuse writer from overflowPool when writers stack is empty")

		_, err = got.Write(payload)
		require.NoError(t, err)
		require.NoError(t, got.Close())

		r, err := comp.Decompress(&buf)
		require.NoError(t, err)
		decoded, err := io.ReadAll(r)
		require.NoError(t, err)
		require.NoError(t, r.(io.Closer).Close())
		assert.Equal(t, payload, decoded)
	})
}

func TestGzipConcurrentCompressDecompress(t *testing.T) {
	setPersistGzipFeatureGateForTest(t, true)
	comp := &gzipCompressor{
		writers:     make([]*gzipWriter, 0, 4),
		maxRetained: 4,
	}
	payload := bytes.Repeat([]byte("concurrent-gzip-check-"), 256)

	var wg sync.WaitGroup
	const workers = 16
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for range 25 {
				var buf bytes.Buffer
				w, err := comp.Compress(&buf)
				if !assert.NoError(t, err) {
					return
				}
				_, err = w.Write(payload)
				if !assert.NoError(t, err) {
					return
				}
				if !assert.NoError(t, w.Close()) {
					return
				}
				r, err := comp.Decompress(&buf)
				if !assert.NoError(t, err) {
					return
				}
				got, err := io.ReadAll(r)
				if !assert.NoError(t, err) {
					return
				}
				if !assert.NoError(t, r.(io.Closer).Close()) {
					return
				}
				assert.Equal(t, payload, got)
			}
		}()
	}
	wg.Wait()
}
