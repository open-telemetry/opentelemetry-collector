// Copyright The OpenTelemetry Authors
// Copyright 2017 gRPC authors.
// SPDX-License-Identifier: Apache-2.0

// Package snappy registers a gRPC snappy compressor compatible with the
// collector's configgrpc package.
package snappy // import "go.opentelemetry.io/collector/internal/grpccompression/snappy"

import (
	"io"
	"sync"

	snappylib "github.com/golang/snappy"
	"google.golang.org/grpc/encoding"
)

// Name is the content-coding used for snappy-compressed gRPC payloads.
const Name = "snappy"

func init() {
	registerCompressor(encoding.GetCompressor, encoding.RegisterCompressor)
}

func registerCompressor(get func(string) encoding.Compressor, register func(encoding.Compressor)) bool {
	if get(Name) != nil {
		return false
	}

	register(newCompressor())
	return true
}

type compressor struct {
	poolCompressor   sync.Pool
	poolDecompressor sync.Pool
}

func newCompressor() *compressor {
	c := &compressor{}
	c.poolCompressor.New = func() any {
		return snappylib.NewBufferedWriter(io.Discard)
	}
	return c
}

func (c *compressor) Compress(w io.Writer) (io.WriteCloser, error) {
	z, ok := c.poolCompressor.Get().(*snappylib.Writer)
	if !ok {
		z = snappylib.NewBufferedWriter(w)
	} else {
		z.Reset(w)
	}

	return &writer{
		Writer: z,
		pool:   &c.poolCompressor,
	}, nil
}

func (c *compressor) Decompress(r io.Reader) (io.Reader, error) {
	z, ok := c.poolDecompressor.Get().(*snappylib.Reader)
	if !ok {
		return &reader{
			Reader: snappylib.NewReader(r),
			pool:   &c.poolDecompressor,
		}, nil
	}

	z.Reset(r)
	return &reader{
		Reader: z,
		pool:   &c.poolDecompressor,
	}, nil
}

func (c *compressor) Name() string {
	return Name
}

type writer struct {
	*snappylib.Writer
	pool *sync.Pool
	once sync.Once
}

func (z *writer) Close() error {
	err := z.Writer.Close()
	z.release()
	return err
}

// release returns the underlying writer to the pool exactly once per wrapper.
func (z *writer) release() {
	z.once.Do(func() {
		z.pool.Put(z.Writer)
	})
}

type reader struct {
	*snappylib.Reader
	pool *sync.Pool
	once sync.Once
}

func (z *reader) Read(p []byte) (int, error) {
	n, err := z.Reader.Read(p)
	if err != nil {
		z.release()
	}
	return n, err
}

// release returns the underlying reader to the pool exactly once per wrapper.
func (z *reader) release() {
	z.once.Do(func() {
		z.pool.Put(z.Reader)
	})
}
