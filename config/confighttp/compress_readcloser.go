// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"fmt"
	"io"
)

// compressReadCloser couples the original compressed reader
// and the compression reader to ensure that the original body
// is correctly closed to ensure resources are freed.
type compressReadCloser struct {
	io.Reader
	orig io.ReadCloser
}

var (
	_ io.Reader = (*compressReadCloser)(nil)
	_ io.Closer = (*compressReadCloser)(nil)
)

func (crc *compressReadCloser) Close() error {
	return crc.orig.Close()
}

// panicRecoverReadCloser wraps an io.ReadCloser and recovers from
// panics during Read, converting them to errors. Some decompression
// libraries (e.g., zstd) may panic on malformed input instead of
// returning an error.
type panicRecoverReadCloser struct {
	inner io.ReadCloser
}

func (pr *panicRecoverReadCloser) Read(p []byte) (n int, err error) {
	defer func() {
		if r := recover(); r != nil {
			n = 0
			err = fmt.Errorf("decompression panic: %v", r)
		}
	}()
	return pr.inner.Read(p)
}

func (pr *panicRecoverReadCloser) Close() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("decompression close panic: %v", r)
		}
	}()
	return pr.inner.Close()
}
