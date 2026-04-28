// Copyright The OpenTelemetry Authors
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
		return &writer{
			Writer: snappylib.NewBufferedWriter(io.Discard),
			pool:   &c.poolCompressor,
		}
	}
	return c
}

func (c *compressor) Compress(w io.Writer) (io.WriteCloser, error) {
	z := c.poolCompressor.Get().(*writer)
	z.once = sync.Once{}
	z.Reset(w)
	return z, nil
}

func (c *compressor) Decompress(r io.Reader) (io.Reader, error) {
	z, ok := c.poolDecompressor.Get().(*reader)
	if !ok {
		return &reader{
			Reader: snappylib.NewReader(r),
			pool:   &c.poolDecompressor,
		}, nil
	}

	z.once = sync.Once{}
	z.Reset(r)
	return z, nil
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
	z.once.Do(func() {
		z.pool.Put(z)
	})
	return err
}

type reader struct {
	*snappylib.Reader
	pool *sync.Pool
	once sync.Once
}

func (z *reader) Read(p []byte) (int, error) {
	n, err := z.Reader.Read(p)
	if err != nil {
		z.once.Do(func() {
			z.pool.Put(z)
		})
	}
	return n, err
}
