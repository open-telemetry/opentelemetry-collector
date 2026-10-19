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

func TestSizeCacheMerge(t *testing.T) {
	t.Run("both_dimensions_cached_on_both", func(t *testing.T) {
		c1 := NewSizeCache()
		c1.Update(SizerTypeItems, 10)
		c1.SizeOf(SizerTypeBytes, func() int { return 100 })

		c2 := NewSizeCache()
		c2.Update(SizerTypeItems, 20)
		c2.SizeOf(SizerTypeBytes, func() int { return 200 })

		c1.Merge(c2)

		itemsCalls, bytesCalls := 0, 0
		assert.Equal(t, 30, c1.SizeOf(SizerTypeItems, func() int { itemsCalls++; return -1 }))
		assert.Equal(t, 300, c1.SizeOf(SizerTypeBytes, func() int { bytesCalls++; return -1 }))
		assert.Equal(t, 0, itemsCalls, "items must be served from cache")
		assert.Equal(t, 0, bytesCalls, "bytes must be served from cache")
	})

	t.Run("one_dimension_uncached_on_one_side", func(t *testing.T) {
		c1 := NewSizeCache()
		c1.Update(SizerTypeItems, 10)
		// bytes uncached on c1

		c2 := NewSizeCache()
		c2.Update(SizerTypeItems, 20)
		c2.SizeOf(SizerTypeBytes, func() int { return 200 })

		c1.Merge(c2)

		itemsCalls, bytesCalls := 0, 0
		assert.Equal(t, 30, c1.SizeOf(SizerTypeItems, func() int { itemsCalls++; return -1 }))
		assert.Equal(t, 0, itemsCalls, "items must be served from cache")

		assert.Equal(t, 500, c1.SizeOf(SizerTypeBytes, func() int { bytesCalls++; return 500 }))
		assert.Equal(t, 1, bytesCalls, "bytes must be recomputed on cache miss")
	})

	t.Run("bytes_cached_items_uncached_on_one_side", func(t *testing.T) {
		c1 := NewSizeCache()
		c1.Update(SizerTypeBytes, 100)

		c2 := NewSizeCache()
		c2.Update(SizerTypeBytes, 200)

		c1.Merge(c2)

		itemsCalls, bytesCalls := 0, 0
		assert.Equal(t, 300, c1.SizeOf(SizerTypeBytes, func() int { bytesCalls++; return -1 }))
		assert.Equal(t, 0, bytesCalls, "bytes must be served from cache")

		assert.Equal(t, 15, c1.SizeOf(SizerTypeItems, func() int { itemsCalls++; return 15 }))
		assert.Equal(t, 1, itemsCalls, "items must be recomputed on cache miss")
	})

	t.Run("zero_size_preserved_in_merge", func(t *testing.T) {
		c1 := NewSizeCache()
		c1.Update(SizerTypeItems, 0)
		c1.Update(SizerTypeBytes, 100)

		c2 := NewSizeCache()
		c2.Update(SizerTypeItems, 5)
		c2.Update(SizerTypeBytes, 0)

		// c1.Update cleared items, so re-populate items as 0
		c1.itemsSize = 0
		c2.itemsSize = 5

		c1.Merge(c2)

		itemsCalls, bytesCalls := 0, 0
		assert.Equal(t, 5, c1.SizeOf(SizerTypeItems, func() int { itemsCalls++; return -1 }))
		assert.Equal(t, 100, c1.SizeOf(SizerTypeBytes, func() int { bytesCalls++; return -1 }))
		assert.Equal(t, 0, itemsCalls)
		assert.Equal(t, 0, bytesCalls)
	})
}

func TestSizeCacheReset(t *testing.T) {
	c := NewSizeCache()
	c.SizeOf(SizerTypeItems, func() int { return 10 })
	c.SizeOf(SizerTypeBytes, func() int { return 100 })

	c.Reset()

	itemsCalls, bytesCalls := 0, 0
	assert.Equal(t, 0, c.SizeOf(SizerTypeItems, func() int { itemsCalls++; return -1 }))
	assert.Equal(t, 0, c.SizeOf(SizerTypeBytes, func() int { bytesCalls++; return -1 }))
	assert.Equal(t, 0, itemsCalls, "items must be cached at 0")
	assert.Equal(t, 0, bytesCalls, "bytes must be cached at 0")
}
