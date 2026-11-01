// Copyright The OpenTelemetry Authors
// Copyright 2020 Mostyn Bramley-Moore.
// SPDX-License-Identifier: Apache-2.0

// Package zstd registers a gRPC zstd compressor compatible with the
// collector's configgrpc package.
package zstd // import "go.opentelemetry.io/collector/config/configgrpc/internal/grpccompression/zstd"

import (
	"io"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/klauspost/compress/zstd"
	"google.golang.org/grpc/encoding"
)

// Name is the content-coding used for zstd-compressed gRPC payloads.
const Name = "zstd"

// defaultEncoderWindow is the largest window a zstd encoder with default
// settings declares. An encoder that does not buffer the whole payload before
// writing the frame header cannot shrink the window to the content size, so it
// declares its configured window instead: klauspost/compress v1.20.3 declares
// 8MiB for a 1MiB payload. Frames from such encoders must keep decoding, so the
// decoder bound never goes below this value.
const defaultEncoderWindow = 8 << 20

// maxWindow bounds the window the decoder allocates. A frame header declares
// its window before any output is produced, and the library default allows
// windows up to 512MiB, so without a bound a tiny crafted frame can force a huge
// allocation. It starts at defaultEncoderWindow, and configgrpc raises it to
// the largest max_recv_msg_size_mib any peer configured with SetMaxRecvMsgSize.
var maxWindow atomic.Uint64

var encoderOptions = []zstd.EOption{
	// The default zstd window size is larger than typical OTLP payloads and
	// can drive unnecessary memory usage.
	zstd.WithWindowSize(512 * 1024),
}

var decoderOptions = []zstd.DOption{
	// Disabling async decode avoids extra decoder resources that are not
	// useful for single-request gRPC decompression.
	zstd.WithDecoderConcurrency(1),
}

var zstdNewWriter = zstd.NewWriter

var zstdNewReader = zstd.NewReader

var zstdEncoderReset = func(enc *zstd.Encoder, w io.Writer) {
	enc.Reset(w)
}

var zstdDecoderReset = func(dec *zstd.Decoder, r io.Reader) error {
	return dec.Reset(r)
}

func init() {
	maxWindow.Store(defaultEncoderWindow)
	registerCompressor(encoding.GetCompressor, encoding.RegisterCompressor)
}

func registerCompressor(get func(string) encoding.Compressor, register func(encoding.Compressor)) bool {
	if get(Name) != nil {
		return false
	}

	register(&compressor{})
	return true
}

// SetMaxRecvMsgSize records the largest message a peer accepts, so the decoder
// allocates a window large enough for frames that carry one. grpc-go bounds the
// decoded message at the configured max_recv_msg_size_mib, so a larger window
// cannot yield more output than the peer reads.
//
// The compressor is registered process-wide, so the bound is the largest size
// any peer configured: a peer with a small limit must not tighten the bound for
// one that accepts larger messages.
func SetMaxRecvMsgSize(size int) {
	if size <= 0 {
		return
	}
	for {
		current := maxWindow.Load()
		if uint64(size) <= current || maxWindow.CompareAndSwap(current, uint64(size)) {
			return
		}
	}
}

// DecoderMaxWindow returns the largest window the decoder will allocate.
func DecoderMaxWindow() uint64 {
	return maxWindow.Load()
}

type compressor struct {
	encoderPool sync.Pool
	decoderPool sync.Pool
}

func (c *compressor) Compress(w io.Writer) (io.WriteCloser, error) {
	enc, ok := c.encoderPool.Get().(*zstd.Encoder)
	if !ok {
		var err error
		enc, err = zstdNewWriter(w, encoderOptions...)
		if err != nil {
			return nil, err
		}
	} else {
		zstdEncoderReset(enc, w)
	}

	wrapper := &encoderWrapper{
		Encoder: enc,
		pool:    &c.encoderPool,
	}
	runtime.SetFinalizer(wrapper, (*encoderWrapper).finalize)
	return wrapper, nil
}

func (c *compressor) Decompress(r io.Reader) (io.Reader, error) {
	window := maxWindow.Load()

	dec := c.takeDecoder(window)
	if dec == nil {
		opts := make([]zstd.DOption, 0, len(decoderOptions)+1)
		opts = append(opts, decoderOptions...)
		opts = append(opts, zstd.WithDecoderMaxWindow(window))

		var err error
		dec, err = zstdNewReader(r, opts...)
		if err != nil {
			return nil, err
		}
	} else if err := zstdDecoderReset(dec, r); err != nil {
		c.putDecoder(dec, window)
		return nil, err
	}

	wrapper := &decoderWrapper{
		Decoder: dec,
		pool:    &c.decoderPool,
		window:  window,
	}
	runtime.SetFinalizer(wrapper, (*decoderWrapper).finalize)
	return wrapper, nil
}

func (c *compressor) Name() string {
	return Name
}

// takeDecoder returns a pooled decoder created with the given window bound. A
// decoder created under a different bound is dropped rather than reused, so
// raising the bound never leaves a decoder that rejects a frame it should
// accept.
func (c *compressor) takeDecoder(window uint64) *zstd.Decoder {
	pd, ok := c.decoderPool.Get().(*pooledDecoder)
	if !ok {
		return nil
	}
	if pd.window != window {
		pd.decoder.Close()
		return nil
	}
	return pd.decoder
}

func (c *compressor) putDecoder(dec *zstd.Decoder, window uint64) {
	c.decoderPool.Put(&pooledDecoder{decoder: dec, window: window})
}

// pooledDecoder records the window bound its decoder was created with.
type pooledDecoder struct {
	decoder *zstd.Decoder
	window  uint64
}

type encoderWrapper struct {
	*zstd.Encoder
	pool *sync.Pool
	once sync.Once
}

func (w *encoderWrapper) Close() error {
	err := w.Encoder.Close()
	w.release(func() {
		w.pool.Put(w.Encoder)
	})
	return err
}

func (w *encoderWrapper) finalize() {
	w.release(func() {
		zstdEncoderReset(w.Encoder, nil)
		w.pool.Put(w.Encoder)
	})
}

func (w *encoderWrapper) release(fn func()) {
	w.once.Do(fn)
}

type decoderWrapper struct {
	*zstd.Decoder
	pool     *sync.Pool
	window   uint64
	once     sync.Once
	released atomic.Bool
}

func (d *decoderWrapper) Read(p []byte) (int, error) {
	// A released decoder is back in the pool and may already belong to another
	// message, so never read through it again.
	if d.released.Load() {
		return 0, zstd.ErrDecoderClosed
	}
	n, err := d.Decoder.Read(p)
	if err != nil {
		d.release()
	}
	return n, err
}

// Close returns the decoder to the pool. grpc-go closes a decompressor reader
// that implements io.Closer as soon as it is done with a message, including
// when it stops reading early at max_recv_msg_size, so the decoder and its
// window are reused instead of being held until the finalizer runs.
func (d *decoderWrapper) Close() error {
	d.release()
	return nil
}

func (d *decoderWrapper) finalize() {
	d.release()
}

func (d *decoderWrapper) release() {
	d.once.Do(func() {
		d.released.Store(true)
		if err := zstdDecoderReset(d.Decoder, nil); err == nil {
			d.pool.Put(&pooledDecoder{decoder: d.Decoder, window: d.window})
		}
	})
}
