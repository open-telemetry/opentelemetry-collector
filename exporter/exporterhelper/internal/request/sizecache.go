// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"

// uncachedSize marks a dimension that has not been computed yet.
const uncachedSize = -1

// SizeCache caches the size of a request per sizer type.
// Sizes for SizerTypeRequests are not cached because they are constant.
type SizeCache struct {
	itemsSize int
	bytesSize int
}

func NewSizeCache() SizeCache {
	return SizeCache{itemsSize: uncachedSize, bytesSize: uncachedSize}
}

// SizeOf returns the size of the request for a given SizerType, and calls compute on a miss.
func (c *SizeCache) SizeOf(szt SizerType, compute func() int) int {
	switch szt {
	case SizerTypeItems:
		if c.itemsSize == uncachedSize {
			c.itemsSize = compute()
		}
		return c.itemsSize
	case SizerTypeBytes:
		if c.bytesSize == uncachedSize {
			c.bytesSize = compute()
		}
		return c.bytesSize
	default:
		return compute()
	}
}

// Update records the size for a SizerType after the request data changed, and drops
// values of other types, which the caller did not maintain.
func (c *SizeCache) Update(szt SizerType, size int) {
	c.itemsSize, c.bytesSize = uncachedSize, uncachedSize
	switch szt {
	case SizerTypeItems:
		c.itemsSize = size
	case SizerTypeBytes:
		c.bytesSize = size
	}
}
