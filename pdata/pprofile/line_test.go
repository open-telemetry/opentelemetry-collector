// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLineSliceEqual(t *testing.T) {
	for _, tt := range []struct {
		name string
		orig LineSlice
		dest LineSlice
		want bool
	}{
		{
			name: "with empty slices",
			orig: NewLineSlice(),
			dest: NewLineSlice(),
			want: true,
		},
		{
			name: "with non-empty equal slices",
			orig: func() LineSlice {
				ls := NewLineSlice()
				ls.AppendEmpty().SetLine(1)
				return ls
			}(),
			dest: func() LineSlice {
				ls := NewLineSlice()
				ls.AppendEmpty().SetLine(1)
				return ls
			}(),
			want: true,
		},
		{
			name: "with different lengths",
			orig: func() LineSlice {
				ls := NewLineSlice()
				ls.AppendEmpty()
				return ls
			}(),
			dest: NewLineSlice(),
			want: false,
		},
		{
			name: "with non-equal slices",
			orig: func() LineSlice {
				ls := NewLineSlice()
				ls.AppendEmpty().SetLine(2)
				return ls
			}(),
			dest: func() LineSlice {
				ls := NewLineSlice()
				ls.AppendEmpty().SetLine(1)
				return ls
			}(),
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want {
				assert.True(t, tt.orig.Equal(tt.dest))
			} else {
				assert.False(t, tt.orig.Equal(tt.dest))
			}
		})
	}
}

func TestLineEqual(t *testing.T) {
	for _, tt := range []struct {
		name string
		orig Line
		dest Line
		want bool
	}{
		{
			name: "with empty lines",
			orig: NewLine(),
			dest: NewLine(),
			want: true,
		},
		{
			name: "with non-empty lines",
			orig: func() Line {
				l := NewLine()
				l.SetColumn(1)
				l.SetFunctionIndex(2)
				l.SetLine(3)

				return l
			}(),
			dest: func() Line {
				l := NewLine()
				l.SetColumn(1)
				l.SetFunctionIndex(2)
				l.SetLine(3)

				return l
			}(),
			want: true,
		},
		{
			name: "with non-equal column",
			orig: func() Line {
				l := NewLine()
				l.SetColumn(1)
				l.SetFunctionIndex(2)
				l.SetLine(3)

				return l
			}(),
			dest: func() Line {
				l := NewLine()
				l.SetColumn(2)
				l.SetFunctionIndex(2)
				l.SetLine(3)

				return l
			}(),
			want: false,
		},
		{
			name: "with non-equal function index",
			orig: func() Line {
				l := NewLine()
				l.SetColumn(1)
				l.SetFunctionIndex(2)
				l.SetLine(3)

				return l
			}(),
			dest: func() Line {
				l := NewLine()
				l.SetColumn(1)
				l.SetFunctionIndex(3)
				l.SetLine(3)

				return l
			}(),
			want: false,
		},
		{
			name: "with non-equal line",
			orig: func() Line {
				l := NewLine()
				l.SetColumn(1)
				l.SetFunctionIndex(2)
				l.SetLine(3)

				return l
			}(),
			dest: func() Line {
				l := NewLine()
				l.SetColumn(1)
				l.SetFunctionIndex(2)
				l.SetLine(4)

				return l
			}(),
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want {
				assert.True(t, tt.orig.Equal(tt.dest))
			} else {
				assert.False(t, tt.orig.Equal(tt.dest))
			}
		})
	}
}
