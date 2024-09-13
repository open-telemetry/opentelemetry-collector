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
	"github.com/pierrec/lz4/v4"

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
	// Concurrency 1 disables async decoding via goroutines. This is useful to reduce memory usage and isn't a bottleneck for compression using sync.Pool.
	zStdPool                  = &compressor{pool: sync.Pool{New: func() any { zw, _ := zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1)); return zw }}}
	_        writeCloserReset = (*zlib.Writer)(nil)
	zLibPool                  = &compressor{pool: sync.Pool{New: func() any { return zlib.NewWriter(nil) }}}
	_        writeCloserReset = (*lz4.Writer)(nil)
	lz4Pool                   = &compressor{pool: sync.Pool{New: func() any {
		lz := lz4.NewWriter(nil)
		// Setting concurrency to 1 to disable async decoding by goroutines. This will reduce the overall memory footprint and pool
		_ = lz.Apply(lz4.ConcurrencyOption(1))
		return lz
	}}}
)

type compressor struct {
	pool sync.Pool
}

// writerFactory defines writer field in CompressRoundTripper.
// The validity of input is already checked when NewCompressRoundTripper was called in confighttp,
func newCompressor(compressionType configcompression.Type) (*compressor, error) {
	switch compressionType {
	case configcompression.TypeGzip:
		return gZipPool, nil
	case configcompression.TypeSnappy:
		return snappyPool, nil
	case configcompression.TypeZstd:
		return zStdPool, nil
	case configcompression.TypeZlib, configcompression.TypeDeflate:
		return zLibPool, nil
	case configcompression.TypeLz4:
		return lz4Pool, nil
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
