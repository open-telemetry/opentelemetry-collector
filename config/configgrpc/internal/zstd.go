// Copyright The OpenTelemetry Authors
// Copyright 2017 gRPC authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/config/configgrpc/internal"

import (
	"errors"
	"io"
	"sync"

	"github.com/klauspost/compress/zstd"
	"google.golang.org/grpc/encoding"
)

const ZstdName = "zstd"

func init() {
	encoding.RegisterCompressor(NewZstdCodec())
}

type writer struct {
	*zstd.Encoder
	pool *sync.Pool
}

func NewZstdCodec() encoding.Compressor {
	c := &compressor{}
	c.poolCompressor.New = func() any {
		zw, _ := zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1), zstd.WithWindowSize(512*1024))
		return &writer{Encoder: zw, pool: &c.poolCompressor}
	}
	return c
}

func (c *compressor) Compress(w io.Writer) (io.WriteCloser, error) {
	z := c.poolCompressor.Get().(*writer)
	z.Encoder.Reset(w)
	return z, nil
}

func (z *writer) Close() error {
	defer z.pool.Put(z)
	return z.Encoder.Close()
}

type reader struct {
	*zstd.Decoder
	pool *sync.Pool
}

func (c *compressor) Decompress(r io.Reader) (io.Reader, error) {
	z, inPool := c.poolDecompressor.Get().(*reader)
	if !inPool {
		newZ, err := zstd.NewReader(r)
		if err != nil {
			return nil, err
		}
		return &reader{Decoder: newZ, pool: &c.poolDecompressor}, nil
	}
	if err := z.Reset(r); err != nil {
		c.poolDecompressor.Put(z)
		return nil, err
	}
	return z, nil
}

func (z *reader) Read(p []byte) (n int, err error) {
	n, err = z.Decoder.Read(p)
	if errors.Is(err, io.EOF) {
		z.pool.Put(z)
	}
	return n, err
}

func (c *compressor) Name() string {
	return ZstdName
}

type compressor struct {
	poolCompressor   sync.Pool
	poolDecompressor sync.Pool
}
