// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testutil"
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

func TestLineSwitchDictionary(t *testing.T) {
	for _, tt := range []struct {
		name string
		line Line

		src ProfilesDictionary
		dst ProfilesDictionary

		wantLine       Line
		wantDictionary ProfilesDictionary
		wantErr        error
	}{
		{
			name: "with an empty line",
			line: NewLine(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantLine:       NewLine(),
			wantDictionary: NewProfilesDictionary(),
		},
		{
			name: "with an existing function",
			line: func() Line {
				l := NewLine()
				l.SetFunctionIndex(1)
				return l
			}(),

			src: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "test")

				d.FunctionTable().AppendEmpty()
				f := d.FunctionTable().AppendEmpty()
				f.SetNameStrindex(1)
				return d
			}(),
			dst: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo")

				d.FunctionTable().AppendEmpty()
				d.FunctionTable().AppendEmpty()
				return d
			}(),

			wantLine: func() Line {
				l := NewLine()
				l.SetFunctionIndex(2)
				return l
			}(),
			wantDictionary: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo", "test")

				d.FunctionTable().AppendEmpty()
				d.FunctionTable().AppendEmpty()
				f := d.FunctionTable().AppendEmpty()
				f.SetNameStrindex(2)
				return d
			}(),
		},
		{
			name: "with a function index that does not match anything",
			line: func() Line {
				l := NewLine()
				l.SetFunctionIndex(1)
				return l
			}(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantLine: func() Line {
				l := NewLine()
				l.SetFunctionIndex(1)
				return l
			}(),
			wantDictionary: NewProfilesDictionary(),
			wantErr:        errors.New("invalid function index 1"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			line := tt.line
			dst := tt.dst
			err := line.switchDictionary(tt.src, dst)

			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, tt.wantErr, err)
			}

			assert.Equal(t, tt.wantLine, line)
			assert.Equal(t, tt.wantDictionary, dst)
		})
	}
}

func BenchmarkLineSwitchDictionary(b *testing.B) {
	testutil.SkipMemoryBench(b)

	l := NewLine()
	l.SetFunctionIndex(1)

	src := NewProfilesDictionary()
	src.StringTable().Append("", "test")

	src.FunctionTable().AppendEmpty()
	src.FunctionTable().AppendEmpty().SetNameStrindex(1)

	b.ReportAllocs()

	for b.Loop() {
		b.StopTimer()
		dst := NewProfilesDictionary()
		b.StartTimer()

		_ = l.switchDictionary(src, dst)
	}
}

func buildLine(col int64, funcIdx int32, line int64) Line {
	l := NewLine()
	l.SetColumn(col)
	l.SetFunctionIndex(funcIdx)
	l.SetLine(line)

	return l
}
