// Copyright The OpenTelemetry Authors
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

func init() {
	if encoding.GetCompressor(Name) != nil {
		return
	}

	encoding.RegisterCompressor(&compressor{})
}

type compressor struct {
	encoderPool sync.Pool
	decoderPool sync.Pool
}

func (c *compressor) Compress(w io.Writer) (io.WriteCloser, error) {
	enc, ok := c.encoderPool.Get().(*zstd.Encoder)
	if !ok {
		var err error
		enc, err = zstd.NewWriter(w, encoderOptions...)
		if err != nil {
			return nil, err
		}
	} else {
		enc.Reset(w)
	}

	wrapper := &encoderWrapper{
		Encoder: enc,
		pool:    &c.encoderPool,
	}
	runtime.SetFinalizer(wrapper, func(ew *encoderWrapper) {
		ew.release(func() {
			ew.Encoder.Reset(nil)
			ew.pool.Put(ew.Encoder)
		})
	})
	return wrapper, nil
}

func (c *compressor) Decompress(r io.Reader) (io.Reader, error) {
	dec, ok := c.decoderPool.Get().(*zstd.Decoder)
	if !ok {
		var err error
		dec, err = zstd.NewReader(r, decoderOptions...)
		if err != nil {
			return nil, err
		}
	} else if err := dec.Reset(r); err != nil {
		c.decoderPool.Put(dec)
		return nil, err
	}

	wrapper := &decoderWrapper{
		Decoder: dec,
		pool:    &c.decoderPool,
	}
	runtime.SetFinalizer(wrapper, func(dw *decoderWrapper) {
		dw.release()
	})
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

func (d *decoderWrapper) release() {
	d.once.Do(func() {
		if err := d.Decoder.Reset(nil); err == nil {
			d.pool.Put(d.Decoder)
		}
	})
}
