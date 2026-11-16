// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configgrpc // import "go.opentelemetry.io/collector/config/configgrpc"

import (
	"compress/gzip"
	"io"
	"runtime"
	"sync"

	"google.golang.org/grpc/encoding"
	// Import grpc's gzip package first so its init() runs before ours and our
	// RegisterCompressor call wraps the default sync.Pool-only compressor.
	grpcgzip "google.golang.org/grpc/encoding/gzip"

	"go.opentelemetry.io/collector/config/configgrpc/internal/metadata"
)

func init() {
	fallback := encoding.GetCompressor(grpcgzip.Name)
	encoding.RegisterCompressor(newGzipCompressor(fallback))
}

type gzipCompressor struct {
	fallback     encoding.Compressor
	mu           sync.Mutex
	writers      []*gzipWriter
	maxRetained  int
	overflowPool sync.Pool
	readerPool   sync.Pool
}

func newGzipCompressor(fallback encoding.Compressor) *gzipCompressor {
	maxRetained := max(runtime.GOMAXPROCS(0), 1)
	return &gzipCompressor{
		fallback:    fallback,
		writers:     make([]*gzipWriter, 0, maxRetained),
		maxRetained: maxRetained,
	}
}

type gzipWriter struct {
	*gzip.Writer
	c *gzipCompressor
}

func (c *gzipCompressor) Compress(w io.Writer) (io.WriteCloser, error) {
	if !metadata.PkgConfiggrpcPersistGzipCompressorsFeatureGate.IsEnabled() && c.fallback != nil {
		return c.fallback.Compress(w)
	}

	c.mu.Lock()
	if n := len(c.writers); n > 0 {
		z := c.writers[n-1]
		c.writers[n-1] = nil
		c.writers = c.writers[:n-1]
		c.mu.Unlock()
		z.Writer.Reset(w)
		return z, nil
	}
	c.mu.Unlock()

	if v := c.overflowPool.Get(); v != nil {
		z := v.(*gzipWriter)
		z.Writer.Reset(w)
		return z, nil
	}
	return &gzipWriter{
		Writer: gzip.NewWriter(w),
		c:      c,
	}, nil
}

func (z *gzipWriter) Close() error {
	err := z.Writer.Close()
	z.c.Put(z)
	return err
}

func (c *gzipCompressor) Put(z *gzipWriter) {
	c.mu.Lock()
	if len(c.writers) < c.maxRetained {
		c.writers = append(c.writers, z)
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
	c.overflowPool.Put(z)
}

var _ io.Closer = (*gzipReader)(nil)

type gzipReader struct {
	*gzip.Reader
	pool *sync.Pool
}

func (c *gzipCompressor) Decompress(r io.Reader) (io.Reader, error) {
	if !metadata.PkgConfiggrpcPersistGzipCompressorsFeatureGate.IsEnabled() && c.fallback != nil {
		return c.fallback.Decompress(r)
	}

	z, inPool := c.readerPool.Get().(*gzipReader)
	if !inPool {
		newZ, err := gzip.NewReader(r)
		if err != nil {
			return nil, err
		}
		return &gzipReader{Reader: newZ, pool: &c.readerPool}, nil
	}
	if err := z.Reset(r); err != nil {
		c.readerPool.Put(z)
		return nil, err
	}
	return z, nil
}

func (r *gzipReader) Close() error {
	defer r.pool.Put(r)
	return r.Reader.Close()
}

func (*gzipCompressor) Name() string {
	return grpcgzip.Name
}
