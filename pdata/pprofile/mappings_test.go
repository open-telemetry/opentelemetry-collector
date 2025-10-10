// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetMapping(t *testing.T) {
	table := NewMappingSlice()
	m := NewMapping()
	m.SetMemoryLimit(1)
	m2 := NewMapping()
	m2.SetMemoryLimit(2)
	loc := NewLocation()

	// Put a first mapping
	require.NoError(t, SetMapping(table, loc, m))
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), loc.MappingIndex())

	// Put the same mapping
	// This should be a no-op.
	require.NoError(t, SetMapping(table, loc, m))
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), loc.MappingIndex())

	// Set a new mapping
	// This sets the index and adds to the table.
	require.NoError(t, SetMapping(table, loc, m2))
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), loc.MappingIndex()) //nolint:gosec // G115

	// Set an existing mapping
	require.NoError(t, SetMapping(table, loc, m))
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(0), loc.MappingIndex())
	// Set another existing mapping
	require.NoError(t, SetMapping(table, loc, m2))
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), loc.MappingIndex()) //nolint:gosec // G115
}

func TestSetMappingCurrentTooHigh(t *testing.T) {
	table := NewMappingSlice()
	loc := NewLocation()
	loc.SetMappingIndex(42)

	err := SetMapping(table, loc, NewMapping())
	require.Error(t, err)
	assert.Equal(t, 0, table.Len())
	assert.Equal(t, int32(42), loc.MappingIndex())
}

func BenchmarkSetMapping(b *testing.B) {
	for _, bb := range []struct {
		name    string
		mapping Mapping

		runBefore func(*testing.B, MappingSlice, Location)
	}{
		{
			name:    "with a new mapping",
			mapping: NewMapping(),
		},
		{
			name: "with an existing mapping",
			mapping: func() Mapping {
				m := NewMapping()
				m.SetMemoryLimit(1)
				return m
			}(),

			runBefore: func(_ *testing.B, table MappingSlice, _ Location) {
				m := table.AppendEmpty()
				m.SetMemoryLimit(1)
			},
		},
		{
			name:    "with a duplicate mapping",
			mapping: NewMapping(),

			runBefore: func(_ *testing.B, table MappingSlice, obj Location) {
				require.NoError(b, SetMapping(table, obj, NewMapping()))
			},
		},
		{
			name: "with a hundred mappings to loop through",
			mapping: func() Mapping {
				m := NewMapping()
				m.SetMemoryLimit(1)
				return m
			}(),

			runBefore: func(_ *testing.B, table MappingSlice, _ Location) {
				for i := range 100 {
					m := table.AppendEmpty()
					m.SetMemoryLimit(uint64(i)) //nolint:gosec // overflow checked
				}
			},
		},
	} {
		b.Run(bb.name, func(b *testing.B) {
			table := NewMappingSlice()
			obj := NewLocation()

			if bb.runBefore != nil {
				bb.runBefore(b, table, obj)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = SetMapping(table, obj, bb.mapping)
			}
		})
	}
}
