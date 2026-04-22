// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package snappy

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/encoding"
)

func TestRegisteredCompression(t *testing.T) {
	comp := encoding.GetCompressor(Name)
	require.NotNil(t, comp)
	assert.Equal(t, Name, comp.Name())

	for _, tt := range []struct {
		name    string
		payload []byte
	}{
		{name: "empty", payload: nil},
		{name: "small", payload: []byte("message request snappy")},
		{name: "large", payload: bytes.Repeat([]byte("snappy payload "), 1024)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := roundTripCompression(t, comp, tt.payload)
			if len(tt.payload) == 0 {
				assert.Empty(t, got)
				return
			}
			assert.Equal(t, tt.payload, got)
		})
	}
}

func TestDecoderReturnedToPoolOnReadCompletion(t *testing.T) {
	comp := encoding.GetCompressor(Name)
	require.NotNil(t, comp)

	c, ok := comp.(*compressor)
	require.True(t, ok)

	compressed := compressPayload(t, comp, []byte("snappy pool check"))

	r, err := c.Decompress(bytes.NewReader(compressed))
	require.NoError(t, err)

	_, err = io.ReadAll(r)
	require.NoError(t, err)

	pooled, ok := c.poolDecompressor.Get().(*reader)
	require.True(t, ok, "expected decoder to be returned to the pool")
	c.poolDecompressor.Put(pooled)
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
