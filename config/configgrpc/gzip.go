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
	if !metadata.PkgConfiggrpcPersistGzipCompressorsFeatureGate.IsEnabled() {
		return c.fallback.Compress(w)
	}

	c.mu.Lock()
	if n := len(c.writers); n > 0 {
		z := c.writers[n-1]
		c.writers[n-1] = nil
		c.writers = c.writers[:n-1]
		c.mu.Unlock()
		z.Reset(w)
		return z, nil
	}
	c.mu.Unlock()

	if v := c.overflowPool.Get(); v != nil {
		z := v.(*gzipWriter)
		z.Reset(w)
		return z, nil
	}
	return &gzipWriter{
		Writer: gzip.NewWriter(w),
		c:      c,
	}, nil
}

func (z *gzipWriter) Close() error {
	err := z.Writer.Close()
	z.c.mu.Lock()
	if len(z.c.writers) < z.c.maxRetained {
		z.c.writers = append(z.c.writers, z)
		z.c.mu.Unlock()
		return err
	}
	z.c.mu.Unlock()
	z.c.overflowPool.Put(z)
	return err
}

func (c *gzipCompressor) Decompress(r io.Reader) (io.Reader, error) {
	return c.fallback.Decompress(r)
}

func (*gzipCompressor) Name() string {
	return grpcgzip.Name
}
