// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testutil"
)

func TestSetFunction(t *testing.T) {
	table := NewFunctionSlice()
	f := NewFunction()
	f.SetNameStrindex(1)
	f2 := NewFunction()
	f2.SetNameStrindex(2)

	// Put a first function
	idx, err := SetFunction(table, f)
	require.NoError(t, err)
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), idx)

	// Put the same function
	// This should be a no-op.
	idx, err = SetFunction(table, f)
	require.NoError(t, err)
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), idx)

	// Set a new function
	// This sets the index and adds to the table.
	idx, err = SetFunction(table, f2)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), idx)

	// Set an existing function
	idx, err = SetFunction(table, f)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(0), idx)
	// Set another existing function
	idx, err = SetFunction(table, f2)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), idx)
}

func BenchmarkSetFunction(b *testing.B) {
	testutil.SkipMemoryBench(b)
	for _, bb := range []struct {
		name string
		fn   Function

		runBefore func(*testing.B, FunctionSlice)
	}{
		{
			name: "with a new function",
			fn:   NewFunction(),
		},
		{
			name: "with an existing function",
			fn: func() Function {
				f := NewFunction()
				f.SetNameStrindex(1)
				return f
			}(),

			runBefore: func(_ *testing.B, table FunctionSlice) {
				f := table.AppendEmpty()
				f.SetNameStrindex(1)
			},
		},
		{
			name: "with a duplicate function",
			fn:   NewFunction(),

			runBefore: func(b *testing.B, table FunctionSlice) {
				_, err := SetFunction(table, NewFunction())
				require.NoError(b, err)
			},
		},
		{
			name: "with a hundred functions to loop through",
			fn: func() Function {
				f := NewFunction()
				f.SetNameStrindex(1)
				return f
			}(),

			runBefore: func(_ *testing.B, table FunctionSlice) {
				for i := range 100 {
					f := table.AppendEmpty()
					f.SetNameStrindex(int32(i))
				}
			},
		},
	} {
		b.Run(bb.name, func(b *testing.B) {
			table := NewFunctionSlice()

			if bb.runBefore != nil {
				bb.runBefore(b, table)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = SetFunction(table, bb.fn)
			}
		})
	}
}
