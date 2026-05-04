// Copyright The OpenTelemetry Authors
// Copyright 2021 gRPC authors.
// SPDX-License-Identifier: Apache-2.0

package snappy

import (
	"bytes"
	"io"
	"runtime"
	"testing"

	snappylib "github.com/golang/snappy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/encoding"
)

func TestRegisteredCompression(t *testing.T) {
	for _, tt := range []struct {
		name    string
		payload []byte
	}{
		{name: "empty", payload: nil},
		{name: "small", payload: []byte("message request snappy")},
		{name: "large", payload: bytes.Repeat([]byte("snappy payload "), 1024)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			comp := newCompressor()
			got := roundTripCompression(t, comp, tt.payload)
			if len(tt.payload) == 0 {
				assert.Empty(t, got)
				return
			}
			assert.Equal(t, tt.payload, got)
		})
	}
}

func TestInitRegistration(t *testing.T) {
	comp := encoding.GetCompressor(Name)
	require.NotNil(t, comp)
	assert.Equal(t, Name, comp.Name())
}

func TestRegisterCompressor(t *testing.T) {
	t.Run("registers when missing", func(t *testing.T) {
		var registered encoding.Compressor
		ok := registerCompressor(
			func(string) encoding.Compressor { return nil },
			func(c encoding.Compressor) { registered = c },
		)
		require.True(t, ok)
		require.NotNil(t, registered)
		assert.Equal(t, Name, registered.Name())
	})

	t.Run("skips when already registered", func(t *testing.T) {
		var called bool
		ok := registerCompressor(
			func(string) encoding.Compressor { return newCompressor() },
			func(encoding.Compressor) { called = true },
		)
		require.False(t, ok)
		assert.False(t, called)
	})
}

func TestDecompressReusesPooledReader(t *testing.T) {
	c := newCompressor()
	seed := snappylib.NewReader(bytes.NewReader(nil))

	c.poolDecompressor.New = func() any { return seed }

	compressed := compressPayload(t, c, []byte("snappy reuse check"))

	r, err := c.Decompress(bytes.NewReader(compressed))
	require.NoError(t, err)
	got, ok := r.(*reader)
	require.True(t, ok)
	assert.Same(t, seed, got.Reader)

	out, err := io.ReadAll(got)
	require.NoError(t, err)
	assert.Equal(t, []byte("snappy reuse check"), out)
}

func TestCompressReusesPooledWriter(t *testing.T) {
	c := newCompressor()
	seed := snappylib.NewBufferedWriter(io.Discard)
	c.poolCompressor.New = func() any { return seed }
	payload := []byte("snappy writer reuse check")

	var buf bytes.Buffer
	w, err := c.Compress(&buf)
	require.NoError(t, err)
	got, ok := w.(*writer)
	require.True(t, ok)
	assert.Same(t, seed, got.Writer)
	_, err = got.Write(payload)
	require.NoError(t, err)
	require.NoError(t, got.Close())

	r, err := c.Decompress(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, payload, out)
}

func compressPayload(t *testing.T, comp encoding.Compressor, payload []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	w, err := comp.Compress(&buf)
	require.NoError(t, err)

	_, err = w.Write(payload)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	return buf.Bytes()
}

func roundTripCompression(t *testing.T, comp encoding.Compressor, payload []byte) []byte {
	t.Helper()

	compressed := compressPayload(t, comp, payload)

	r, err := comp.Decompress(bytes.NewReader(compressed))
	require.NoError(t, err)

	got, err := io.ReadAll(r)
	require.NoError(t, err)
	return got
}

func TestConcurrentWriterReuseDoesNotPanic(t *testing.T) {
	prev := runtime.GOMAXPROCS(runtime.NumCPU())
	defer runtime.GOMAXPROCS(prev)

	comp := newCompressor()
	payload := bytes.Repeat([]byte("snappy payload "), 32)

	const workers = 6
	const iterations = 100

	start := make(chan struct{})
	errCh := make(chan error, workers)

	for range workers {
		go func() {
			<-start
			for range iterations {
				var buf bytes.Buffer
				w, err := comp.Compress(&buf)
				if err != nil {
					errCh <- err
					return
				}

				if _, err = w.Write(payload); err != nil {
					errCh <- err
					return
				}

				if err = w.Close(); err != nil {
					errCh <- err
					return
				}
			}
			errCh <- nil
		}()
	}

	close(start)

	for range workers {
		require.NoError(t, <-errCh)
	}
}

func TestConcurrentReaderReuseDoesNotPanic(t *testing.T) {
	prev := runtime.GOMAXPROCS(runtime.NumCPU())
	defer runtime.GOMAXPROCS(prev)

	comp := newCompressor()
	payload := bytes.Repeat([]byte("snappy payload "), 32)
	compressed := compressPayload(t, comp, payload)

	const workers = 8
	const iterations = 50

	start := make(chan struct{})
	errCh := make(chan error, workers)

	for range workers {
		go func() {
			<-start
			for range iterations {
				r, err := comp.Decompress(bytes.NewReader(compressed))
				if err != nil {
					errCh <- err
					return
				}

				if _, err = io.ReadAll(r); err != nil {
					errCh <- err
					return
				}
			}
			errCh <- nil
		}()
	}

	close(start)

	for range workers {
		require.NoError(t, <-errCh)
	}
}
