// Copyright The OpenTelemetry Authors
// Copyright 2021 gRPC authors.
// SPDX-License-Identifier: Apache-2.0

package zstd

import (
	"bytes"
	"errors"
	"io"
	"runtime"
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
	c.decoderPool.New = func() any { return &pooledDecoder{decoder: seed, window: defaultEncoderWindow} }

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
	c.decoderPool.New = func() any { return &pooledDecoder{decoder: seed, window: defaultEncoderWindow} }

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
		window:  defaultEncoderWindow,
	}
	d.finalize()
	assert.Nil(t, resetTarget)
}

func TestDecompressRejectsOversizedWindow(t *testing.T) {
	// A 10-byte frame whose header declares a 512MiB window: the decoder
	// allocates the window as soon as it parses the header, before it produces
	// any output.
	payload := []byte{
		0x28, 0xb5, 0x2f, 0xfd, // frame magic
		0x00,             // frame header descriptor: no content size, no checksum
		19 << 3,          // window descriptor: 2^29 = 512MiB
		0x09, 0x00, 0x00, // last block, raw, 1 byte
		0x41, // block payload
	}
	var header kzstd.Header
	require.NoError(t, header.Decode(payload))
	require.Equal(t, uint64(512*1024*1024), header.WindowSize)

	c := &compressor{}
	r, err := c.Decompress(bytes.NewReader(payload))
	require.NoError(t, err)

	var out []byte
	allocated := allocatedBytes(func() {
		out, err = io.ReadAll(r)
	})

	require.ErrorIs(t, err, kzstd.ErrWindowSizeExceeded)
	assert.Empty(t, out)
	assert.Less(t, allocated, uint64(4*1024*1024), "decoder must not allocate the window the frame declares")
}

func TestDecompressDecodesFrameFromDefaultEncoder(t *testing.T) {
	// A default-configured encoder that does not buffer the whole payload
	// declares its 8MiB window instead of the content size, so the window bound
	// must stay above the default max receive message size for this frame to
	// keep decoding.
	payload := bytes.Repeat([]byte("collector payload "), 64*1024)
	var buf bytes.Buffer
	enc, err := kzstd.NewWriter(&buf)
	require.NoError(t, err)
	_, err = enc.Write(payload)
	require.NoError(t, err)
	require.NoError(t, enc.Close())

	var header kzstd.Header
	require.NoError(t, header.Decode(buf.Bytes()))
	require.Greater(t, header.WindowSize, uint64(4*1024*1024), "frame must declare a window above grpc's default max receive message size")

	r, err := (&compressor{}).Decompress(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)

	got, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestDecoderWrapperImplementsIOCloser(t *testing.T) {
	c := &compressor{}
	compressed := compressPayload(t, c, []byte("zstd closer check"))

	r, err := c.Decompress(bytes.NewReader(compressed))
	require.NoError(t, err)

	// grpc-go only releases the decoder when it implements io.Closer.
	closer, ok := r.(io.Closer)
	require.True(t, ok, "decoder wrapper must implement io.Closer")
	require.NoError(t, closer.Close())

	// A closed reader must not touch the decoder again: it is back in the pool
	// and may belong to another message.
	_, err = r.Read(make([]byte, 8))
	assert.ErrorIs(t, err, kzstd.ErrDecoderClosed)
}

func TestCloseAfterPartialReadReleasesDecoder(t *testing.T) {
	// grpc-go reads at most max_recv_msg_size+1 bytes and then closes the
	// reader. Without an io.Closer the decoder would be held until the
	// finalizer runs.
	var resetArgs []io.Reader
	oldReset := zstdDecoderReset
	t.Cleanup(func() { zstdDecoderReset = oldReset })
	zstdDecoderReset = func(dec *kzstd.Decoder, r io.Reader) error {
		resetArgs = append(resetArgs, r)
		return oldReset(dec, r)
	}

	c := &compressor{}
	compressed := compressPayload(t, c, bytes.Repeat([]byte("z"), 64*1024))

	r, err := c.Decompress(bytes.NewReader(compressed))
	require.NoError(t, err)
	_, err = io.ReadFull(r, make([]byte, 10))
	require.NoError(t, err)

	closer, ok := r.(io.Closer)
	require.True(t, ok)
	require.NoError(t, closer.Close())

	// Releasing resets the decoder to drop its reader and then puts it back
	// into the pool, so the next message reuses it.
	require.NotEmpty(t, resetArgs)
	assert.Nil(t, resetArgs[len(resetArgs)-1], "Close must reset the decoder before returning it to the pool")

	require.NoError(t, closer.Close(), "Close must be idempotent")
	assert.Len(t, resetArgs, 1, "a second Close must not release the decoder again")
}

func TestDecoderMaxWindow(t *testing.T) {
	old := maxWindow.Load()
	t.Cleanup(func() { maxWindow.Store(old) })
	maxWindow.Store(defaultEncoderWindow)

	assert.Equal(t, uint64(defaultEncoderWindow), DecoderMaxWindow())

	SetMaxRecvMsgSize(32 << 20)
	assert.Equal(t, uint64(32<<20), DecoderMaxWindow())
}

func TestTakeDecoderDropsDecoderBuiltWithAnotherWindow(t *testing.T) {
	stale, err := kzstd.NewReader(bytes.NewReader(nil), decoderOptions...)
	require.NoError(t, err)

	c := &compressor{}
	c.decoderPool.New = func() any { return &pooledDecoder{decoder: stale, window: 1 << 20} }

	// Raising the bound must not hand back a decoder that would reject a frame
	// the new bound allows.
	assert.Nil(t, c.takeDecoder(defaultEncoderWindow))
}

func TestSetMaxRecvMsgSizeRaisesWindowBound(t *testing.T) {
	old := maxWindow.Load()
	t.Cleanup(func() { maxWindow.Store(old) })

	SetMaxRecvMsgSize(0)
	SetMaxRecvMsgSize(-1)
	assert.Equal(t, old, maxWindow.Load(), "non-positive sizes must not change the bound")

	SetMaxRecvMsgSize(64 << 20)
	assert.Equal(t, uint64(64<<20), maxWindow.Load())

	SetMaxRecvMsgSize(4 << 20)
	assert.Equal(t, uint64(64<<20), maxWindow.Load(), "a smaller peer limit must not tighten the bound")
}

// allocatedBytes reports the bytes allocated while fn runs.
func allocatedBytes(fn func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	fn()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
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
