// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestSetLink(t *testing.T) {
	table := NewLinkSlice()
	l := NewLink()
	l.SetTraceID(pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}))
	l2 := NewLink()
	l.SetTraceID(pcommon.TraceID([16]byte{2, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 2}))
	smpl := NewSample()

	// Put a first link
	require.NoError(t, SetLink(table, smpl, l))
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), smpl.LinkIndex())

	// Put the same link
	// This should be a no-op.
	require.NoError(t, SetLink(table, smpl, l))
	assert.Equal(t, 1, table.Len())
	assert.Equal(t, int32(0), smpl.LinkIndex())

	// Set a new link
	// This sets the index and adds to the table.
	require.NoError(t, SetLink(table, smpl, l2))
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), smpl.LinkIndex()) //nolint:gosec // G115

	// Set an existing link
	require.NoError(t, SetLink(table, smpl, l))
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(0), smpl.LinkIndex())
	// Set another existing link
	require.NoError(t, SetLink(table, smpl, l2))
	assert.Equal(t, 2, table.Len())
	assert.Equal(t, int32(table.Len()-1), smpl.LinkIndex()) //nolint:gosec // G115
}

func TestSetLinkCurrentTooHigh(t *testing.T) {
	table := NewLinkSlice()
	smpl := NewSample()
	smpl.SetLinkIndex(42)

	err := SetLink(table, smpl, NewLink())
	require.Error(t, err)
	assert.Equal(t, 0, table.Len())
	assert.Equal(t, int32(42), smpl.LinkIndex())
}

func BenchmarkSetLink(b *testing.B) {
	for _, bb := range []struct {
		name string
		link Link

		runBefore func(*testing.B, LinkSlice, Sample)
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

			runBefore: func(_ *testing.B, table LinkSlice, _ Sample) {
				l := table.AppendEmpty()
				l.SetTraceID(pcommon.NewTraceIDEmpty())
			},
		},
		{
			name: "with a duplicate link",
			link: NewLink(),

			runBefore: func(_ *testing.B, table LinkSlice, obj Sample) {
				require.NoError(b, SetLink(table, obj, NewLink()))
			},
		},
		{
			name: "with a hundred links to loop through",
			link: func() Link {
				l := NewLink()
				l.SetTraceID(pcommon.NewTraceIDEmpty())
				return l
			}(),

			runBefore: func(_ *testing.B, table LinkSlice, _ Sample) {
				for range 100 {
					table.AppendEmpty()
				}
			},
		},
	} {
		b.Run(bb.name, func(b *testing.B) {
			table := NewLinkSlice()
			obj := NewSample()

			if bb.runBefore != nil {
				bb.runBefore(b, table, obj)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = SetLink(table, obj, bb.link)
			}
		})
	}
}
