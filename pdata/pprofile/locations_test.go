// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testutil"
)

func TestFromLocationIndices(t *testing.T) {
	table := NewLocationSlice()
	table.AppendEmpty().SetAddress(1)
	table.AppendEmpty().SetAddress(2)

	stack := NewStack()
	locs := FromLocationIndices(table, stack)
	assert.Equal(t, locs, NewLocationSlice())

	// Add a location
	stack.LocationIndices().Append(0)
	locs = FromLocationIndices(table, stack)

	tLoc := NewLocationSlice()
	tLoc.AppendEmpty().SetAddress(1)
	assert.Equal(t, tLoc, locs)

	// Add another location
	stack.LocationIndices().Append(1)

	locs = FromLocationIndices(table, stack)
	assert.Equal(t, table, locs)
}

func TestSetLocation(t *testing.T) {
	table := NewLocationSlice()
	l := NewLocation()
	l.SetAddress(1)
	l2 := NewLocation()
	l2.SetAddress(2)

	// Put a first value
	idx, err := SetLocation(table, l)
	require.NoError(t, err)
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), idx)

	// Put the same string
	// This should be a no-op.
	idx, err = SetLocation(table, l)
	require.NoError(t, err)
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), idx)

	// Set a new value
	// This sets the index and adds to the table.
	idx, err = SetLocation(table, l2)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), idx)

	// Set an existing value
	idx, err = SetLocation(table, l)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(0), idx)
	// Set another existing value
	idx, err = SetLocation(table, l2)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), idx)
}

func BenchmarkFromLocationIndices(b *testing.B) {
	table := NewLocationSlice()

	for i := range 100 {
		table.AppendEmpty().SetAddress(uint64(i))
	}

	obj := NewStack()
	for i := range int32(50) {
		obj.LocationIndices().Append(2*i + 1)
	}

	b.ReportAllocs()

	for b.Loop() {
		_ = FromLocationIndices(table, obj)
	}
}

func BenchmarkSetLocation(b *testing.B) {
	testutil.SkipMemoryBench(b)
	for _, bb := range []struct {
		name     string
		location Location

		runBefore func(*testing.B, LocationSlice)
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

			runBefore: func(_ *testing.B, table LocationSlice) {
				l := table.AppendEmpty()
				l.SetAddress(1)
			},
		},
		{
			name:     "with a duplicate location",
			location: NewLocation(),

			runBefore: func(_ *testing.B, table LocationSlice) {
				_, err := SetLocation(table, NewLocation())
				require.NoError(b, err)
			},
		},
		{
			name: "with a hundred locations to loop through",
			location: func() Location {
				l := NewLocation()
				l.SetMappingIndex(1)
				return l
			}(),

			runBefore: func(_ *testing.B, table LocationSlice) {
				for i := range 100 {
					l := table.AppendEmpty()
					l.SetAddress(uint64(i))
				}

				l := table.AppendEmpty()
				l.SetMappingIndex(1)
			},
		},
	} {
		b.Run(bb.name, func(b *testing.B) {
			table := NewLocationSlice()

			if bb.runBefore != nil {
				bb.runBefore(b, table)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = SetLocation(table, bb.location)
			}
		})
	}
}
