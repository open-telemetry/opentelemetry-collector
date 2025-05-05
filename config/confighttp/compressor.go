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

type compressor struct {
	pool sync.Pool
}

type compressorMap map[compressionMapKey]*compressor

type compressionMapKey struct {
	compressionType   configcompression.Type
	compressionParams configcompression.CompressionParams
}

var (
	compressorPools   = make(compressorMap)
	compressorPoolsMu sync.Mutex
)

// writerFactory defines writer field in CompressRoundTripper.
// The validity of input is already checked when NewCompressRoundTripper was called in confighttp,
func newCompressor(compressionType configcompression.Type, compressionParams configcompression.CompressionParams) (*compressor, error) {
	compressorPoolsMu.Lock()
	defer compressorPoolsMu.Unlock()
	mapKey := compressionMapKey{compressionType, compressionParams}
	c, ok := compressorPools[mapKey]
	if ok {
		return c, nil
	}

	f, err := newWriteCloserResetFunc(compressionType, compressionParams)
	if err != nil {
		return nil, err
	}
	c = &compressor{pool: sync.Pool{New: func() any { return f() }}}
	compressorPools[mapKey] = c
	return c, nil
}

func newWriteCloserResetFunc(compressionType configcompression.Type, compressionParams configcompression.CompressionParams) (func() writeCloserReset, error) {
	switch compressionType {
	case configcompression.TypeGzip:
		return func() writeCloserReset {
			w, _ := gzip.NewWriterLevel(nil, int(compressionParams.Level))
			return w
		}, nil
	case configcompression.TypeSnappyFramed:
		if !enableFramedSnappy.IsEnabled() {
			return nil, errors.New("x-snappy-framed is not enabled")
		}
		return func() writeCloserReset {
			return snappy.NewBufferedWriter(nil)
		}, nil
	case configcompression.TypeSnappy:
		if !enableFramedSnappy.IsEnabled() {
			// If framed snappy feature gate is not enabled, we keep the current behavior
			// where the 'Content-Encoding: snappy' is compressed as the framed snappy format.
			return func() writeCloserReset {
				return snappy.NewBufferedWriter(nil)
			}, nil
		}
		return func() writeCloserReset {
			// If framed snappy feature gate is enabled, we use the correct behavior
			// where the 'Content-Encoding: snappy' is compressed as the block snappy format.
			return &rawSnappyWriter{}
		}, nil
	case configcompression.TypeZstd:
		level := zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(int(compressionParams.Level)))
		return func() writeCloserReset {
			zw, _ := zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1), level)
			return zw
		}, nil
	case configcompression.TypeZlib, configcompression.TypeDeflate:
		return func() writeCloserReset {
			w, _ := zlib.NewWriterLevel(nil, int(compressionParams.Level))
			return w
		}, nil
	case configcompression.TypeLz4:
		return func() writeCloserReset {
			lz := lz4.NewWriter(nil)
			_ = lz.Apply(lz4.ConcurrencyOption(1))
			return lz
		}, nil
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

// rawSnappyWriter buffers all writes and, on Close,
// compresses the data as a raw snappy block (non-framed)
// and writes the compressed bytes to the underlying writer.
type rawSnappyWriter struct {
	buffer bytes.Buffer
	w      io.Writer
	closed bool
}

// Write buffers the data.
func (w *rawSnappyWriter) Write(p []byte) (int, error) {
	return w.buffer.Write(p)
}

// Close compresses the buffered data in one shot using snappy.Encode,
// writes the compressed block to the underlying writer, and marks the writer as closed.
func (w *rawSnappyWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	// Compress the buffered uncompressed bytes.
	compressed := snappy.Encode(nil, w.buffer.Bytes())
	_, err := w.w.Write(compressed)
	return err
}

// Reset sets a new underlying writer, resets the buffer and the closed flag.
func (w *rawSnappyWriter) Reset(newWriter io.Writer) {
	w.buffer.Reset()
	w.w = newWriter
	w.closed = false
}
