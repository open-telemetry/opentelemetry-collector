// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package reclaim

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlignHelpers(t *testing.T) {
	t.Parallel()

	assert.Equal(t, uint64(0), alignDown(0, 4096))
	assert.Equal(t, uint64(4096), alignDown(4096, 4096))
	assert.Equal(t, uint64(4096), alignDown(8191, 4096))
	assert.Equal(t, uint64(123), alignDown(123, 0))

	v, ok := alignUp(0, 4096)
	assert.True(t, ok)
	assert.Equal(t, uint64(0), v)

	v, ok = alignUp(1, 4096)
	assert.True(t, ok)
	assert.Equal(t, uint64(4096), v)

	v, ok = alignUp(4096, 4096)
	assert.True(t, ok)
	assert.Equal(t, uint64(4096), v)

	v, ok = alignUp(123, 0)
	assert.True(t, ok)
	assert.Equal(t, uint64(123), v)

	_, ok = alignUp(math.MaxUint64-10, 4096)
	assert.False(t, ok)
}

func TestSubtractIntervals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		base     []interval
		excluded []interval
		want     []interval
	}{
		{
			name:     "merges_overlapping_base_and_punches_hole",
			base:     []interval{{start: 0x3000, end: 0x8000}, {start: 0x1000, end: 0x4000}},
			excluded: []interval{{start: 0x3000, end: 0x5000}},
			want:     []interval{{start: 0x1000, end: 0x3000}, {start: 0x5000, end: 0x8000}},
		},
		{
			name:     "ignores_empty_and_disjoint_exclusions",
			base:     []interval{{start: 0x1000, end: 0x3000}, {start: 0x5000, end: 0x5000}},
			excluded: []interval{{start: 0x4000, end: 0x4000}, {start: 0x6000, end: 0x7000}},
			want:     []interval{{start: 0x1000, end: 0x3000}},
		},
		{
			name:     "complete_removal",
			base:     []interval{{start: 0x2000, end: 0x4000}},
			excluded: []interval{{start: 0x1000, end: 0x5000}},
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, subtractIntervals(tt.base, tt.excluded))
		})
	}
}

func TestIntersectInterval(t *testing.T) {
	t.Parallel()

	ov, ok := intersectInterval(interval{start: 10, end: 30}, interval{start: 20, end: 40})
	assert.True(t, ok)
	assert.Equal(t, interval{start: 20, end: 30}, ov)

	_, ok = intersectInterval(interval{start: 10, end: 20}, interval{start: 20, end: 30})
	assert.False(t, ok)
}
