// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSizeCacheComputesOncePerDimension(t *testing.T) {
	c := NewSizeCache()
	itemsCalls, bytesCalls := 0, 0
	items := func() int { itemsCalls++; return 7 }
	bytes := func() int { bytesCalls++; return 42 }

	assert.Equal(t, 7, c.SizeOf(SizerTypeItems, items))
	assert.Equal(t, 7, c.SizeOf(SizerTypeItems, items))
	assert.Equal(t, 1, itemsCalls)

	// Caching one dimension must not populate or disturb the other.
	assert.Equal(t, 42, c.SizeOf(SizerTypeBytes, bytes))
	assert.Equal(t, 42, c.SizeOf(SizerTypeBytes, bytes))
	assert.Equal(t, 1, bytesCalls)
	assert.Equal(t, 7, c.SizeOf(SizerTypeItems, items))
	assert.Equal(t, 1, itemsCalls)
}

func TestSizeCacheUpdateDropsOtherDimension(t *testing.T) {
	c := NewSizeCache()
	c.SizeOf(SizerTypeItems, func() int { return 7 })
	c.SizeOf(SizerTypeBytes, func() int { return 42 })

	c.Update(SizerTypeItems, 9)
	assert.Equal(t, 9, c.SizeOf(SizerTypeItems, func() int { return -1 }))
	// Bytes were not maintained by the caller, so they must be recomputed.
	assert.Equal(t, 43, c.SizeOf(SizerTypeBytes, func() int { return 43 }))

	c.Update(SizerTypeBytes, 50)
	assert.Equal(t, 50, c.SizeOf(SizerTypeBytes, func() int { return -1 }))
	assert.Equal(t, 10, c.SizeOf(SizerTypeItems, func() int { return 10 }))
}

// A zero size is a real value, not a cache miss.
func TestSizeCacheCachesZero(t *testing.T) {
	c := NewSizeCache()
	c.Update(SizerTypeBytes, 0)
	calls := 0
	assert.Equal(t, 0, c.SizeOf(SizerTypeBytes, func() int { calls++; return 1 }))
	assert.Equal(t, 0, calls)
}

// Sizes for SizerTypeRequests are constant, so they are never cached, and an
// Update for it must not leave a stale value in either types.
func TestSizeCacheIgnoresRequestsSizerType(t *testing.T) {
	c := NewSizeCache()
	c.SizeOf(SizerTypeItems, func() int { return 7 })

	calls := 0
	assert.Equal(t, 1, c.SizeOf(SizerTypeRequests, func() int { calls++; return 1 }))
	assert.Equal(t, 1, c.SizeOf(SizerTypeRequests, func() int { calls++; return 1 }))
	assert.Equal(t, 2, calls)

	c.Update(SizerTypeRequests, 1)
	assert.Equal(t, 8, c.SizeOf(SizerTypeItems, func() int { return 8 }))
}
