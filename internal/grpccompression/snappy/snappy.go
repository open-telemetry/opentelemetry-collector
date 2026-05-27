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

var (
	writerPool = sync.Pool{
		New: func() any {
			return snappylib.NewBufferedWriter(io.Discard)
		},
	}
	readerPool = sync.Pool{
		New: func() any {
			return snappylib.NewReader(nil)
		},
	}
)

type compressor struct{}

func newCompressor() *compressor {
	return &compressor{}
}

func (*compressor) Compress(w io.Writer) (io.WriteCloser, error) {
	z := writerPool.Get().(*snappylib.Writer)
	z.Reset(w)
	return &writer{Writer: z}, nil
}

func (c *compressor) Decompress(r io.Reader) (io.Reader, error) {
	z := readerPool.Get().(*snappylib.Reader)
	z.Reset(r)
	return &reader{Reader: z}, nil
}

func (c *compressor) Name() string {
	return Name
}

type writer struct {
	*snappylib.Writer
}

func (z *writer) Close() error {
	err := z.Writer.Close()
	writerPool.Put(z.Writer)
	z.Writer = nil
	return err
}

type reader struct {
	*snappylib.Reader
}

func (z *reader) Read(p []byte) (int, error) {
	n, err := z.Reader.Read(p)
	if err != nil {
		readerPool.Put(z.Reader)
		z.Reader = nil
	}
	return n, err
}
