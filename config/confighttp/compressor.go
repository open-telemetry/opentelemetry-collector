// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"fmt"
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

type compressorMap struct {
	pools map[string]*compressor
}

type compressor struct {
	pool sync.Pool
}

var (
	compressorPools                   = &compressorMap{pools: make(map[string]*compressor)}
	gZipCompressor                    = &compressor{}
	snappyCompressor                  = &compressor{}
	zstdCompressor                    = &compressor{}
	zlibCompressor                    = &compressor{}
	lz4Compressor                     = &compressor{}
	_                writeCloserReset = (*gzip.Writer)(nil)
	_                writeCloserReset = (*snappy.Writer)(nil)
	_                writeCloserReset = (*zstd.Encoder)(nil)
	_                writeCloserReset = (*zlib.Writer)(nil)
	_                writeCloserReset = (*lz4.Writer)(nil)
)

// writerFactory defines writer field in CompressRoundTripper.
// The validity of input is already checked when NewCompressRoundTripper was called in confighttp,
func newCompressor(compressionType configcompression.TypeWithLevel) (*compressor, error) {
	mapKey := fmt.Sprintf("%s/%d", compressionType.Type, compressionType.Level)
	var exists bool
	switch compressionType.Type {
	case configcompression.TypeGzip:
		gZipCompressor, exists = compressorPools.pools[mapKey]
		if exists {
			return gZipCompressor, nil
		}
		gZipCompressor = &compressor{}
		gZipCompressor.pool = sync.Pool{New: func() any { w, _ := gzip.NewWriterLevel(nil, int(compressionType.Level)); return w }}
		compressorPools.pools[mapKey] = gZipCompressor
		return gZipCompressor, nil
	case configcompression.TypeSnappy:
		if snappyCompressor.pool.Get() == nil {
			snappyCompressor.pool = sync.Pool{New: func() any { return snappy.NewBufferedWriter(nil) }}
			return snappyCompressor, nil
		}
		return snappyCompressor, nil
	case configcompression.TypeZstd:
		zstdCompressor, exists = compressorPools.pools[mapKey]
		compression := zstd.EncoderLevelFromZstd(int(compressionType.Level))
		encoderLevel := zstd.WithEncoderLevel(compression)
		if exists {
			return zstdCompressor, nil
		}
		zstdCompressor = &compressor{}
		zstdCompressor.pool = sync.Pool{New: func() any { zw, _ := zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1), encoderLevel); return zw }}
		return zstdCompressor, nil
	case configcompression.TypeZlib, configcompression.TypeDeflate:
		zlibCompressor, exists = compressorPools.pools[mapKey]
		if exists {
			return zlibCompressor, nil
		}
		zlibCompressor = &compressor{}
		zlibCompressor.pool = sync.Pool{New: func() any { w, _ := zlib.NewWriterLevel(nil, int(compressionType.Level)); return w }}
		compressorPools.pools[mapKey] = zlibCompressor
		return zlibCompressor, nil
	case configcompression.TypeLz4:
		if lz4Compressor.pool.Get() == nil {
			lz4Compressor.pool = sync.Pool{New: func() any { lz := lz4.NewWriter(nil); _ = lz.Apply(lz4.ConcurrencyOption(1)); return lz }}
			return lz4Compressor, nil
		}
		return lz4Compressor, nil
	}
	return nil, errors.New("unsupported compression type")
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
