// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zstd

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"testing"

	kzstd "github.com/klauspost/compress/zstd"
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
		{name: "small", payload: []byte("message request zstd")},
		{name: "large", payload: bytes.Repeat([]byte("zstd payload "), 4096)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			comp := &compressor{}
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
			func(string) encoding.Compressor { return &compressor{} },
			func(encoding.Compressor) { called = true },
		)
		require.False(t, ok)
		assert.False(t, called)
	})
}

func TestCompressReusesPooledEncoder(t *testing.T) {
	c := &compressor{}
	seed, err := kzstd.NewWriter(io.Discard, encoderOptions...)
	require.NoError(t, err)
	c.encoderPool.New = func() any { return seed }

	old := zstdEncoderReset
	t.Cleanup(func() { zstdEncoderReset = old })
	var resetTarget io.Writer
	zstdEncoderReset = func(enc *kzstd.Encoder, w io.Writer) {
		assert.Same(t, seed, enc)
		resetTarget = w
		enc.Reset(w)
	}

	var buf bytes.Buffer
	w, err := c.Compress(&buf)
	require.NoError(t, err)
	_, ok := w.(*encoderWrapper)
	require.True(t, ok)
	assert.Same(t, &buf, resetTarget)
	_, err = w.Write([]byte("zstd pool check"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
}

func TestDecompressReusesPooledDecoder(t *testing.T) {
	c := &compressor{}
	seed, err := kzstd.NewReader(bytes.NewReader(nil), decoderOptions...)
	require.NoError(t, err)
	c.decoderPool.New = func() any { return seed }

	compressed := compressPayload(t, c, []byte("zstd decode pool check"))
	r, err := c.Decompress(bytes.NewReader(compressed))
	require.NoError(t, err)
	got, ok := r.(*decoderWrapper)
	require.True(t, ok)
	assert.Same(t, seed, got.Decoder)

	out, err := io.ReadAll(got)
	require.NoError(t, err)
	assert.Equal(t, []byte("zstd decode pool check"), out)
}

func TestCompressReturnsEncoderCreationError(t *testing.T) {
	old := zstdNewWriter
	t.Cleanup(func() { zstdNewWriter = old })
	zstdNewWriter = func(io.Writer, ...kzstd.EOption) (*kzstd.Encoder, error) {
		return nil, errors.New("new writer failed")
	}

	_, err := (&compressor{}).Compress(io.Discard)
	require.EqualError(t, err, "new writer failed")
}

func TestEncoderWrapperFinalizeReturnsEncoderToPool(t *testing.T) {
	enc, err := kzstd.NewWriter(io.Discard, encoderOptions...)
	require.NoError(t, err)

	var resetTarget io.Writer
	old := zstdEncoderReset
	t.Cleanup(func() { zstdEncoderReset = old })
	zstdEncoderReset = func(got *kzstd.Encoder, w io.Writer) {
		assert.Same(t, enc, got)
		resetTarget = w
		got.Reset(w)
	}

	w := &encoderWrapper{
		Encoder: enc,
		pool:    &sync.Pool{},
	}
	w.finalize()
	assert.Nil(t, resetTarget)
}

func TestDecompressReturnsDecoderCreationError(t *testing.T) {
	old := zstdNewReader
	t.Cleanup(func() { zstdNewReader = old })
	zstdNewReader = func(io.Reader, ...kzstd.DOption) (*kzstd.Decoder, error) {
		return nil, errors.New("new reader failed")
	}

	_, err := (&compressor{}).Decompress(bytes.NewReader(nil))
	require.EqualError(t, err, "new reader failed")
}

func TestDecompressReturnsDecoderResetError(t *testing.T) {
	old := zstdDecoderReset
	t.Cleanup(func() { zstdDecoderReset = old })
	zstdDecoderReset = func(*kzstd.Decoder, io.Reader) error {
		return errors.New("reset failed")
	}

	c := &compressor{}
	seed, err := kzstd.NewReader(bytes.NewReader(nil), decoderOptions...)
	require.NoError(t, err)
	c.decoderPool.New = func() any { return seed }

	_, err = c.Decompress(bytes.NewReader(nil))
	require.EqualError(t, err, "reset failed")
}

func TestDecoderWrapperFinalizeReturnsDecoderToPool(t *testing.T) {
	dec, err := kzstd.NewReader(bytes.NewReader(nil), decoderOptions...)
	require.NoError(t, err)

	var resetTarget io.Reader = bytes.NewReader(nil)
	old := zstdDecoderReset
	t.Cleanup(func() { zstdDecoderReset = old })
	zstdDecoderReset = func(got *kzstd.Decoder, r io.Reader) error {
		assert.Same(t, dec, got)
		resetTarget = r
		return got.Reset(r)
	}

	d := &decoderWrapper{
		Decoder: dec,
		pool:    &sync.Pool{},
	}
	d.finalize()
	assert.Nil(t, resetTarget)
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
