// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/golang/snappy"
	"github.com/klauspost/compress/zstd"

	"go.opentelemetry.io/collector/config/configcompression"
)

type writeCloserReset interface {
	io.WriteCloser
	Reset(w io.Writer)
}

type compressor struct {
	pool sync.Pool
}

// Gets the compression type and level from the configuration.
func getCompression(compressionField configcompression.Type) (compressionType configcompression.Type, compressionLevel int) {
	parts := strings.Split(string(compressionField), "/")

	compressionLevel = zlib.DefaultCompression
	compressionType = configcompression.Type(parts[0])
	if len(parts) > 1 {
		levelStr := parts[1]
		if level, err := strconv.Atoi(levelStr); err == nil {
			compressionLevel = level
		}
	}
	return compressionType, compressionLevel
}

// writerFactory defines writer field in CompressRoundTripper.
// The validity of input is already checked when NewCompressRoundTripper was called in confighttp,
func newCompressor(compressionType configcompression.Type) (*compressor, error) {
	compressionType, compressionLevel := getCompression(compressionType)

	switch compressionType {
	case configcompression.TypeGzip:
		var _ writeCloserReset = (*gzip.Writer)(nil)
		return &compressor{pool: sync.Pool{New: func() any { w, _ := gzip.NewWriterLevel(nil, compressionLevel); return w }}}, nil
	case configcompression.TypeSnappy:
		var _ writeCloserReset = (*snappy.Writer)(nil)
		return &compressor{pool: sync.Pool{New: func() any { return snappy.NewBufferedWriter(nil) }}}, nil
	case configcompression.TypeZstd:
		var _ writeCloserReset = (*zstd.Encoder)(nil)
		compression := zstd.EncoderLevelFromZstd(compressionLevel)
		encoderLevel := zstd.WithEncoderLevel(compression)
		return &compressor{pool: sync.Pool{New: func() any { zw, _ := zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1), encoderLevel); return zw }}}, nil
	case configcompression.TypeZlib, configcompression.TypeDeflate:
		var _ writeCloserReset = (*zlib.Writer)(nil)
		return &compressor{pool: sync.Pool{New: func() any { w, _ := zlib.NewWriterLevel(nil, compressionLevel); return w }}}, nil
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
