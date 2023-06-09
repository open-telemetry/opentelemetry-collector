// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"io"
	"sync"

	"github.com/golang/snappy"
	"github.com/klauspost/compress/zstd"

	"go.opentelemetry.io/collector/config/configcompression"
)

type writeCloserReset interface {
	io.WriteCloser
	Reset(w io.Writer)
}

var (
	_          writeCloserReset = (*gzip.Writer)(nil)
	gZipPool                    = &compressor{pool: sync.Pool{New: func() any { return gzip.NewWriter(nil) }}}
	_          writeCloserReset = (*snappy.Writer)(nil)
	snappyPool                  = &compressor{pool: sync.Pool{New: func() any { return snappy.NewBufferedWriter(nil) }}}
	_          writeCloserReset = (*zstd.Encoder)(nil)
	zStdPool                    = &compressor{pool: sync.Pool{New: func() any { zw, _ := zstd.NewWriter(nil); return zw }}}
	_          writeCloserReset = (*zlib.Writer)(nil)
	zLibPool                    = &compressor{pool: sync.Pool{New: func() any { return zlib.NewWriter(nil) }}}
)

type compressor struct {
	pool sync.Pool
}

// writerFactory defines writer field in CompressRoundTripper.
// The validity of input is already checked when NewCompressRoundTripper was called in confighttp,
func newCompressor(compressionType configcompression.CompressionType) (*compressor, error) {
	switch compressionType {
	case configcompression.Gzip:
		return gZipPool, nil
	case configcompression.Snappy:
		return snappyPool, nil
	case configcompression.Zstd:
		return zStdPool, nil
	case configcompression.Zlib, configcompression.Deflate:
		return zLibPool, nil
	}
	return nil, errors.New("unsupported compression type, ")
}

func (p *compressor) compress(buf *bytes.Buffer, body io.ReadCloser) error {
	writer := p.pool.Get().(writeCloserReset)
	defer p.pool.Put(writer)
	writer.Reset(buf)

	if body != nil {
		_, copyErr := io.Copy(writer, body)
		closeErr := body.Close()

		if copyErr != nil {
			return copyErr
		}

		if closeErr != nil {
			return closeErr
		}
	}

	return writer.Close()
}
