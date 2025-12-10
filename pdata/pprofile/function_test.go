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

func TestFunctionEqual(t *testing.T) {
	for _, tt := range []struct {
		name string
		orig Function
		dest Function
		want bool
	}{
		{
			name: "empty functions",
			orig: NewFunction(),
			dest: NewFunction(),
			want: true,
		},
		{
			name: "non-empty identical functions",
			orig: buildFunction(1, 2, 3, 4),
			dest: buildFunction(1, 2, 3, 4),
			want: true,
		},
		{
			name: "with different name",
			orig: buildFunction(1, 2, 3, 4),
			dest: buildFunction(2, 2, 3, 4),
			want: false,
		},
		{
			name: "with different system name",
			orig: buildFunction(1, 2, 3, 4),
			dest: buildFunction(1, 3, 3, 4),
			want: false,
		},
		{
			name: "with different file name",
			orig: buildFunction(1, 2, 3, 4),
			dest: buildFunction(1, 2, 4, 4),
			want: false,
		},
		{
			name: "with different start line",
			orig: buildFunction(1, 2, 3, 4),
			dest: buildFunction(1, 2, 3, 5),
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

func TestFunctionSwitchDictionary(t *testing.T) {
	for _, tt := range []struct {
		name     string
		function Function

		src ProfilesDictionary
		dst ProfilesDictionary

		wantFunction   Function
		wantDictionary ProfilesDictionary
		wantErr        error
	}{
		{
			name:     "with an empty key value and unit",
			function: NewFunction(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantFunction:   NewFunction(),
			wantDictionary: NewProfilesDictionary(),
		},
		{
			name: "with an existing name",
			function: func() Function {
				fn := NewFunction()
				fn.SetNameStrindex(1)
				return fn
			}(),

			src: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "test")
				return d
			}(),
			dst: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo")
				return d
			}(),

			wantFunction: func() Function {
				fn := NewFunction()
				fn.SetNameStrindex(2)
				return fn
			}(),
			wantDictionary: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo", "test")
				return d
			}(),
		},
		{
			name: "with a name index that does not match anything",
			function: func() Function {
				fn := NewFunction()
				fn.SetNameStrindex(1)
				return fn
			}(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantFunction: func() Function {
				fn := NewFunction()
				fn.SetNameStrindex(1)
				return fn
			}(),
			wantDictionary: NewProfilesDictionary(),
			wantErr:        errors.New("invalid name index 1"),
		},
		{
			name: "with an existing system name",
			function: func() Function {
				fn := NewFunction()
				fn.SetSystemNameStrindex(1)
				return fn
			}(),

			src: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "test")
				return d
			}(),
			dst: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo")
				return d
			}(),

			wantFunction: func() Function {
				fn := NewFunction()
				fn.SetSystemNameStrindex(2)
				return fn
			}(),
			wantDictionary: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo", "test")
				return d
			}(),
		},
		{
			name: "with a system name index that does not match anything",
			function: func() Function {
				fn := NewFunction()
				fn.SetSystemNameStrindex(1)
				return fn
			}(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantFunction: func() Function {
				fn := NewFunction()
				fn.SetSystemNameStrindex(1)
				return fn
			}(),
			wantDictionary: NewProfilesDictionary(),
			wantErr:        errors.New("invalid system name index 1"),
		},
		{
			name: "with an existing filename",
			function: func() Function {
				fn := NewFunction()
				fn.SetFilenameStrindex(1)
				return fn
			}(),

			src: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "test")
				return d
			}(),
			dst: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo")
				return d
			}(),

			wantFunction: func() Function {
				fn := NewFunction()
				fn.SetFilenameStrindex(2)
				return fn
			}(),
			wantDictionary: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo", "test")
				return d
			}(),
		},
		{
			name: "with a filename index that does not match anything",
			function: func() Function {
				fn := NewFunction()
				fn.SetFilenameStrindex(1)
				return fn
			}(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantFunction: func() Function {
				fn := NewFunction()
				fn.SetFilenameStrindex(1)
				return fn
			}(),
			wantDictionary: NewProfilesDictionary(),
			wantErr:        errors.New("invalid filename index 1"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fn := tt.function
			dst := tt.dst
			err := fn.switchDictionary(tt.src, dst)

			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, tt.wantErr, err)
			}

			assert.Equal(t, tt.wantFunction, fn)
			assert.Equal(t, tt.wantDictionary, dst)
		})
	}
}

func BenchmarkFunctionSwitchDictionary(b *testing.B) {
	testutil.SkipMemoryBench(b)

	fn := NewFunction()
	fn.SetNameStrindex(1)
	fn.SetSystemNameStrindex(2)

	src := NewProfilesDictionary()
	src.StringTable().Append("", "test", "foo")

	b.ReportAllocs()

	for b.Loop() {
		b.StopTimer()
		dst := NewProfilesDictionary()
		dst.StringTable().Append("", "foo")
		b.StartTimer()

		_ = fn.switchDictionary(src, dst)
	}
}

func buildFunction(name, sName, fileName int32, startLine int64) Function {
	f := NewFunction()
	f.SetNameStrindex(name)
	f.SetSystemNameStrindex(sName)
	f.SetFilenameStrindex(fileName)
	f.SetStartLine(startLine)
	return f
}
