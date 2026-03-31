// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testutil"
)

func TestSetMapping(t *testing.T) {
	table := NewMappingSlice()
	m := NewMapping()
	m.SetMemoryLimit(1)
	m2 := NewMapping()
	m2.SetMemoryLimit(2)

	// Put a first mapping
	idx, err := SetMapping(table, m)
	require.NoError(t, err)
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), idx)

	// Put the same mapping
	// This should be a no-op.
	idx, err = SetMapping(table, m)
	require.NoError(t, err)
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), idx)

	// Set a new mapping
	// This sets the index and adds to the table.
	idx, err = SetMapping(table, m2)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), idx)

	// Set an existing mapping
	idx, err = SetMapping(table, m)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(0), idx)
	// Set another existing mapping
	idx, err = SetMapping(table, m2)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), idx)
}

func BenchmarkSetMapping(b *testing.B) {
	testutil.SkipMemoryBench(b)
	for _, bb := range []struct {
		name    string
		mapping Mapping

		runBefore func(*testing.B, MappingSlice)
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

			runBefore: func(_ *testing.B, table MappingSlice) {
				m := table.AppendEmpty()
				m.SetMemoryLimit(1)
			},
		},
		{
			name:    "with a duplicate mapping",
			mapping: NewMapping(),

			runBefore: func(_ *testing.B, table MappingSlice) {
				_, err := SetMapping(table, NewMapping())
				require.NoError(b, err)
			},
		},
		{
			name: "with a hundred mappings to loop through",
			mapping: func() Mapping {
				m := NewMapping()
				m.SetMemoryLimit(1)
				return m
			}(),

			runBefore: func(_ *testing.B, table MappingSlice) {
				for i := range 100 {
					m := table.AppendEmpty()
					m.SetMemoryLimit(uint64(i))
				}
			},
		},
	} {
		b.Run(bb.name, func(b *testing.B) {
			table := NewMappingSlice()

			if bb.runBefore != nil {
				bb.runBefore(b, table)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = SetMapping(table, bb.mapping)
			}
		})
	}
}
