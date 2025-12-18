// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestSetString(t *testing.T) {
	table := pcommon.NewStringSlice()
	v := "test"
	v2 := "test2"

	// Put a first value
	idx, err := SetString(table, v)
	require.NoError(t, err)
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), idx)

	// Put the same string
	// This should be a no-op.
	idx, err = SetString(table, v)
	require.NoError(t, err)
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), idx)

	// Set a new value
	// This sets the index and adds to the table.
	idx, err = SetString(table, v2)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), idx)

	// Set an existing value
	idx, err = SetString(table, v)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(0), idx)
	// Set another existing value
	idx, err = SetString(table, v2)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), idx)
}

func BenchmarkSetString(b *testing.B) {
	for _, bb := range []struct {
		name string
		val  string

		runBefore func(*testing.B, pcommon.StringSlice)
	}{
		{
			name: "with a new value",
			val:  "test",
		},
		{
			name: "with an existing value",
			val:  "test",

			runBefore: func(_ *testing.B, table pcommon.StringSlice) {
				table.Append("test")
			},
		},
		{
			name: "with a duplicate value",
			val:  "test",

			runBefore: func(_ *testing.B, table pcommon.StringSlice) {
				_, err := SetString(table, "test")
				require.NoError(b, err)
			},
		},
		{
			name: "with a hundred values to loop through",
			val:  "test",

			runBefore: func(_ *testing.B, table pcommon.StringSlice) {
				for i := range 100 {
					table.Append(strconv.Itoa(i))
				}
			},
		},
	} {
		b.Run(bb.name, func(b *testing.B) {
			table := pcommon.NewStringSlice()

			if bb.runBefore != nil {
				bb.runBefore(b, table)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = SetString(table, bb.val)
			}
		})
	}
}
