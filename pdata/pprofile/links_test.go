// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestSetLink(t *testing.T) {
	table := NewLinkSlice()
	l := NewLink()
	l.SetTraceID(pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}))
	l2 := NewLink()
	l.SetTraceID(pcommon.TraceID([16]byte{2, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 2}))

	// Put a first link
	idx, err := SetLink(table, l)
	require.NoError(t, err)
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), idx)

	// Put the same link
	// This should be a no-op.
	idx, err = SetLink(table, l)
	require.NoError(t, err)
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), idx)

	// Set a new link
	// This sets the index and adds to the table.
	idx, err = SetLink(table, l2)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), idx)

	// Set an existing link
	idx, err = SetLink(table, l)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(0), idx)
	// Set another existing link
	idx, err = SetLink(table, l2)
	require.NoError(t, err)
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), idx)
}

func BenchmarkSetLink(b *testing.B) {
	testutil.SkipMemoryBench(b)
	for _, bb := range []struct {
		name string
		link Link

		runBefore func(*testing.B, LinkSlice)
	}{
		{
			name: "with a new link",
			link: NewLink(),
		},
		{
			name: "with an existing link",
			link: func() Link {
				l := NewLink()
				l.SetTraceID(pcommon.NewTraceIDEmpty())
				return l
			}(),

			runBefore: func(_ *testing.B, table LinkSlice) {
				l := table.AppendEmpty()
				l.SetTraceID(pcommon.NewTraceIDEmpty())
			},
		},
		{
			name: "with a duplicate link",
			link: NewLink(),

			runBefore: func(b *testing.B, table LinkSlice) {
				_, err := SetLink(table, NewLink())
				require.NoError(b, err)
			},
		},
		{
			name: "with a hundred links to loop through",
			link: func() Link {
				l := NewLink()
				l.SetTraceID(pcommon.NewTraceIDEmpty())
				return l
			}(),

			runBefore: func(_ *testing.B, table LinkSlice) {
				for range 100 {
					table.AppendEmpty()
				}
			},
		},
	} {
		b.Run(bb.name, func(b *testing.B) {
			table := NewLinkSlice()

			if bb.runBefore != nil {
				bb.runBefore(b, table)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, _ = SetLink(table, bb.link)
			}
		})
	}
}
