// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromLocationIndices(t *testing.T) {
	table := NewLocationSlice()
	table.AppendEmpty().SetAddress(1)
	table.AppendEmpty().SetAddress(2)

	prof := NewProfile()
	locs := FromLocationIndices(table, prof)
	assert.Equal(t, locs, NewLocationSlice())

	// Add a location
	prof.LocationIndices().Append(0)
	locs = FromLocationIndices(table, prof)

	tLoc := NewLocationSlice()
	tLoc.AppendEmpty().SetAddress(1)
	assert.Equal(t, tLoc, locs)

	// Add another location
	prof.LocationIndices().Append(1)

	locs = FromLocationIndices(table, prof)
	assert.Equal(t, table, locs)
}

func TestPutLocation(t *testing.T) {
	table := NewLocationSlice()
	l := NewLocation()
	l.SetAddress(1)
	l2 := NewLocation()
	l2.SetAddress(2)
	l3 := NewLocation()
	l3.SetAddress(3)
	l4 := NewLocation()
	l4.SetAddress(4)
	prof := NewProfile()

	// Put a first location
	require.NoError(t, PutLocation(table, prof, l))
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, []int32{0}, prof.LocationIndices().AsRaw())

	// Put the same location
	// This should be a no-op.
	require.NoError(t, PutLocation(table, prof, l))
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, []int32{0}, prof.LocationIndices().AsRaw())

	// Special case: removing and adding again should not change the table as
	// this can lead to multiple identical locations in the table.
	prof.LocationIndices().FromRaw([]int32{})
	require.NoError(t, PutLocation(table, prof, l))
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, []int32{0}, prof.LocationIndices().AsRaw())

	// Put a new location
	// This adds an index and adds to the table.
	require.NoError(t, PutLocation(table, prof, l2))
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, []int32{0, 1}, prof.LocationIndices().AsRaw())

	// Add a negative index to the prof.
	prof.LocationIndices().Append(-1)
	tableLen := table.Len()
	indicesLen := prof.LocationIndices().Len()
	// Try putting a new location, make sure it fails, and that table/indices didn't change.
	require.Error(t, PutLocation(table, prof, l3))
	require.Equal(t, tableLen, table.Len())
	require.Equal(t, indicesLen, prof.LocationIndices().Len())

	// Set the last index to the table length, which is out of range.
	prof.LocationIndices().SetAt(indicesLen-1, int32(tableLen)) //nolint:gosec
	// Try putting a new location, make sure it fails, and that table/indices didn't change.
	require.Error(t, PutLocation(table, prof, l4))
	require.Equal(t, tableLen, table.Len())
	require.Equal(t, indicesLen, prof.LocationIndices().Len())
}

func BenchmarkFromLocationIndices(b *testing.B) {
	table := NewLocationSlice()

	for i := range 10 {
		table.AppendEmpty().SetAddress(uint64(i)) //nolint:gosec // overflow checked
	}

	obj := NewProfile()
	obj.LocationIndices().Append(1, 3, 7)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_ = FromLocationIndices(table, obj)
	}
}

func BenchmarkPutLocation(b *testing.B) {
	for _, bb := range []struct {
		name     string
		location Location

		runBefore func(*testing.B, LocationSlice, locatable)
	}{
		{
			name:     "with a new location",
			location: NewLocation(),
		},
		{
			name: "with an existing location",
			location: func() Location {
				l := NewLocation()
				l.SetAddress(1)
				return l
			}(),

			runBefore: func(_ *testing.B, table LocationSlice, _ locatable) {
				l := table.AppendEmpty()
				l.SetAddress(1)
			},
		},
		{
			name:     "with a duplicate location",
			location: NewLocation(),

			runBefore: func(_ *testing.B, table LocationSlice, obj locatable) {
				require.NoError(b, PutLocation(table, obj, NewLocation()))
			},
		},
		{
			name: "with a hundred locations to loop through",
			location: func() Location {
				l := NewLocation()
				l.SetMappingIndex(1)
				return l
			}(),

			runBefore: func(_ *testing.B, table LocationSlice, _ locatable) {
				for i := range 100 {
					l := table.AppendEmpty()
					l.SetAddress(uint64(i)) //nolint:gosec // overflow checked
				}

				l := table.AppendEmpty()
				l.SetMappingIndex(1)
			},
		},
	} {
		b.Run(bb.name, func(b *testing.B) {
			table := NewLocationSlice()
			obj := NewProfile()

			if bb.runBefore != nil {
				bb.runBefore(b, table, obj)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for n := 0; n < b.N; n++ {
				_ = PutLocation(table, obj, bb.location)
			}
		})
	}
}
