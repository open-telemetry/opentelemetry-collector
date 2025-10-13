// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetStack(t *testing.T) {
	table := NewStackSlice()
	s := NewStack()
	s.LocationIndices().Append(1)
	s2 := NewStack()
	s.LocationIndices().Append(2)
	smpl := NewSample()

	// Put a first stack
	require.NoError(t, SetStack(table, smpl, s))
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), smpl.StackIndex())

	// Put the same stack
	// This should be a no-op.
	require.NoError(t, SetStack(table, smpl, s))
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), smpl.StackIndex())

	// Set a new stack
	// This sets the index and adds to the table.
	require.NoError(t, SetStack(table, smpl, s2))
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), smpl.StackIndex()) //nolint:gosec // G115

	// Set an existing stack
	require.NoError(t, SetStack(table, smpl, s))
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(0), smpl.StackIndex())
	// Set another existing stack
	require.NoError(t, SetStack(table, smpl, s2))
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), smpl.StackIndex()) //nolint:gosec // G115
}

func TestSetStackCurrentTooHigh(t *testing.T) {
	table := NewStackSlice()
	smpl := NewSample()
	smpl.SetStackIndex(42)

	err := SetStack(table, smpl, NewStack())
	require.Error(t, err)
	assert.Equal(t, 0, table.Len())
	assert.Equal(t, int32(42), smpl.StackIndex())
}

func BenchmarkSetStack(b *testing.B) {
	for _, bb := range []struct {
		name  string
		stack Stack

		runBefore func(*testing.B, StackSlice, Sample)
	}{
		{
			name:  "with a new stack",
			stack: NewStack(),
		},
		{
			name: "with an existing stack",
			stack: func() Stack {
				s := NewStack()
				s.LocationIndices().Append(1)
				return s
			}(),

			runBefore: func(_ *testing.B, table StackSlice, _ Sample) {
				s := table.AppendEmpty()
				s.LocationIndices().Append(1)
			},
		},
		{
			name:  "with a duplicate stack",
			stack: NewStack(),

			runBefore: func(_ *testing.B, table StackSlice, obj Sample) {
				require.NoError(b, SetStack(table, obj, NewStack()))
			},
		},
		{
			name: "with a hundred stacks to loop through",
			stack: func() Stack {
				s := NewStack()
				s.LocationIndices().Append(1)
				return s
			}(),

			runBefore: func(_ *testing.B, table StackSlice, _ Sample) {
				for range 100 {
					table.AppendEmpty()
				}
			},
		},
	} {
		b.Run(bb.name, func(b *testing.B) {
			table := NewStackSlice()
			obj := NewSample()

			if bb.runBefore != nil {
				bb.runBefore(b, table, obj)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = SetStack(table, obj, bb.stack)
			}
		})
	}
}
