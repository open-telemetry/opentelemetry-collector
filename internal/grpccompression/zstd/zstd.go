// Copyright The OpenTelemetry Authors
// Copyright 2020 Mostyn Bramley-Moore.
// SPDX-License-Identifier: Apache-2.0

// Package zstd registers a gRPC zstd compressor compatible with the
// collector's configgrpc package.
package zstd // import "go.opentelemetry.io/collector/internal/grpccompression/zstd"

import (
	"io"
	"runtime"
	"sync"

	"github.com/klauspost/compress/zstd"
	"google.golang.org/grpc/encoding"
)

// Name is the content-coding used for zstd-compressed gRPC payloads.
const Name = "zstd"

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
	registerCompressor(encoding.GetCompressor, encoding.RegisterCompressor)
}

func registerCompressor(get func(string) encoding.Compressor, register func(encoding.Compressor)) bool {
	if get(Name) != nil {
		return false
	}

	register(&compressor{})
	return true
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
	dec, ok := c.decoderPool.Get().(*zstd.Decoder)
	if !ok {
		var err error
		dec, err = zstdNewReader(r, decoderOptions...)
		if err != nil {
			return nil, err
		}
	} else if err := zstdDecoderReset(dec, r); err != nil {
		c.decoderPool.Put(dec)
		return nil, err
	}

	wrapper := &decoderWrapper{
		Decoder: dec,
		pool:    &c.decoderPool,
	}
	runtime.SetFinalizer(wrapper, (*decoderWrapper).finalize)
	return wrapper, nil
}

func (c *compressor) Name() string {
	return Name
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
	pool *sync.Pool
	once sync.Once
}

func (d *decoderWrapper) Read(p []byte) (int, error) {
	n, err := d.Decoder.Read(p)
	if err != nil {
		d.release()
	}
	return n, err
}

func (d *decoderWrapper) finalize() {
	d.release()
}

func (d *decoderWrapper) release() {
	d.once.Do(func() {
		if err := zstdDecoderReset(d.Decoder, nil); err == nil {
			d.pool.Put(d.Decoder)
		}
	})
}
