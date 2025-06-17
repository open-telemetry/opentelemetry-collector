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
			orig: buildLine(1, 2, 3),
			dest: buildLine(1, 2, 3),
			want: true,
		},
		{
			name: "with non-equal column",
			orig: buildLine(1, 2, 3),
			dest: buildLine(2, 2, 3),
			want: false,
		},
		{
			name: "with non-equal function index",
			orig: buildLine(1, 2, 3),
			dest: buildLine(1, 3, 3),
			want: false,
		},
		{
			name: "with non-equal line",
			orig: buildLine(1, 2, 3),
			dest: buildLine(1, 2, 4),
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

func buildLine(col int64, funcIdx int32, line int64) Line {
	l := NewLine()
	l.SetColumn(col)
	l.SetFunctionIndex(funcIdx)
	l.SetLine(line)

	return l
}
