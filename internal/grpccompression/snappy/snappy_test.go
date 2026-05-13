// Copyright The OpenTelemetry Authors
// Copyright 2021 gRPC authors.
// SPDX-License-Identifier: Apache-2.0

package snappy

import (
	"bytes"
	"io"
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
	seed := &reader{
		Reader: snappylib.NewReader(bytes.NewReader(nil)),
		pool:   &c.poolDecompressor,
	}
	c.poolDecompressor.New = func() any { return seed }

	compressed := compressPayload(t, c, []byte("snappy reuse check"))

	r, err := c.Decompress(bytes.NewReader(compressed))
	require.NoError(t, err)
	got, ok := r.(*reader)
	require.True(t, ok)
	assert.Same(t, seed, got)

	out, err := io.ReadAll(got)
	require.NoError(t, err)
	assert.Equal(t, []byte("snappy reuse check"), out)
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
