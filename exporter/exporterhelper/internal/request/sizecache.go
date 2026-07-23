// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"

const uncachedSize = -1

// SizeCache caches the byte size of a request.
// Currently, only bytes are cached because are expensive to calculate.
type SizeCache struct {
	bytesSize int
}

func NewSizeCache() SizeCache {
	return SizeCache{bytesSize: uncachedSize}
}

// Get returns the cached byte size if present.
func (sc *SizeCache) Get() (int, bool) {
	return sc.bytesSize, sc.bytesSize != uncachedSize
}

// Set stores the byte size of the request.
func (sc *SizeCache) Set(size int) {
	sc.bytesSize = size
}

// Invalidate clears the cached byte size. Call when the request data changes
// and the new byte size is not known (for example after an items-based merge).
func (sc *SizeCache) Invalidate() {
	sc.bytesSize = uncachedSize
}
