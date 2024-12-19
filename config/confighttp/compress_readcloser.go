// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import "io"

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
