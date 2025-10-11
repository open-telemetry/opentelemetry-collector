// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetFunction(t *testing.T) {
	table := NewFunctionSlice()
	f := NewFunction()
	f.SetNameStrindex(1)
	f2 := NewFunction()
	f2.SetNameStrindex(2)
	li := NewLine()

	// Put a first function
	require.NoError(t, SetFunction(table, li, f))
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), li.FunctionIndex())

	// Put the same function
	// This should be a no-op.
	require.NoError(t, SetFunction(table, li, f))
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), li.FunctionIndex())

	// Set a new function
	// This sets the index and adds to the table.
	require.NoError(t, SetFunction(table, li, f2))
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), li.FunctionIndex()) //nolint:gosec // G115

	// Set an existing function
	require.NoError(t, SetFunction(table, li, f))
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(0), li.FunctionIndex())
	// Set another existing function
	require.NoError(t, SetFunction(table, li, f2))
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), li.FunctionIndex()) //nolint:gosec // G115
}

func TestSetFunctionCurrentTooHigh(t *testing.T) {
	table := NewFunctionSlice()
	li := NewLine()
	li.SetFunctionIndex(42)

	err := SetFunction(table, li, NewFunction())
	require.Error(t, err)
	assert.Equal(t, 0, table.Len())
	assert.Equal(t, int32(42), li.FunctionIndex())
}

func BenchmarkSetFunction(b *testing.B) {
	for _, bb := range []struct {
		name string
		fn   Function

		runBefore func(*testing.B, FunctionSlice, Line)
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

			runBefore: func(_ *testing.B, table FunctionSlice, _ Line) {
				f := table.AppendEmpty()
				f.SetNameStrindex(1)
			},
		},
		{
			name: "with a duplicate function",
			fn:   NewFunction(),

			runBefore: func(_ *testing.B, table FunctionSlice, obj Line) {
				require.NoError(b, SetFunction(table, obj, NewFunction()))
			},
		},
		{
			name: "with a hundred functions to loop through",
			fn: func() Function {
				f := NewFunction()
				f.SetNameStrindex(1)
				return f
			}(),

			runBefore: func(_ *testing.B, table FunctionSlice, _ Line) {
				for i := range 100 {
					f := table.AppendEmpty()
					f.SetNameStrindex(int32(i)) //nolint:gosec // overflow checked
				}
			},
		},
	} {
		b.Run(bb.name, func(b *testing.B) {
			table := NewFunctionSlice()
			obj := NewLine()

			if bb.runBefore != nil {
				bb.runBefore(b, table, obj)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = SetFunction(table, obj, bb.fn)
			}
		})
	}
}
