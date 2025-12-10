// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testutil"
)

func TestSetStack(t *testing.T) {
	table := NewStackSlice()
	s := NewStack()
	s.LocationIndices().Append(1)
	s2 := NewStack()
	s.LocationIndices().Append(2)

	// Put a first stack
	idx, err := SetStack(table, s)
	require.NoError(t, err)
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), idx)

	// Put the same stack
	// This should be a no-op.
	idx, err = SetStack(table, s)
	require.NoError(t, err)
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), idx)

	// Set a new stack
	// This sets the index and adds to the table.
	idx, err = SetStack(table, s2)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), idx)

	// Set an existing stack
	idx, err = SetStack(table, s)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(0), idx)
	// Set another existing stack
	idx, err = SetStack(table, s2)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), idx)
}

func BenchmarkSetStack(b *testing.B) {
	testutil.SkipMemoryBench(b)
	for _, bb := range []struct {
		name  string
		stack Stack

		runBefore func(*testing.B, StackSlice)
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

			runBefore: func(_ *testing.B, table StackSlice) {
				s := table.AppendEmpty()
				s.LocationIndices().Append(1)
			},
		},
		{
			name:  "with a duplicate stack",
			stack: NewStack(),

			runBefore: func(_ *testing.B, table StackSlice) {
				_, err := SetStack(table, NewStack())
				require.NoError(b, err)
			},
		},
		{
			name: "with a hundred stacks to loop through",
			stack: func() Stack {
				s := NewStack()
				s.LocationIndices().Append(1)
				return s
			}(),

			runBefore: func(_ *testing.B, table StackSlice) {
				for range 100 {
					table.AppendEmpty()
				}
			},
		},
	} {
		b.Run(bb.name, func(b *testing.B) {
			table := NewStackSlice()

			if bb.runBefore != nil {
				bb.runBefore(b, table)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = SetStack(table, bb.stack)
			}
		})
	}
}
