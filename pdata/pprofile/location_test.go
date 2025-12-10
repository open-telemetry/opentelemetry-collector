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

func TestLocationEqual(t *testing.T) {
	for _, tt := range []struct {
		name string
		orig Location
		dest Location
		want bool
	}{
		{
			name: "empty locations",
			orig: NewLocation(),
			dest: NewLocation(),
			want: true,
		},
		{
			name: "non-empty locations",
			orig: buildLocation(1, 2, []int32{3}, buildLine(1, 2, 3)),
			dest: buildLocation(1, 2, []int32{3}, buildLine(1, 2, 3)),
			want: true,
		},
		{
			name: "with non-equal mapping index",
			orig: buildLocation(1, 2, []int32{3}, buildLine(1, 2, 3)),
			dest: buildLocation(2, 2, []int32{3}, buildLine(1, 2, 3)),
			want: false,
		},
		{
			name: "with non-equal address",
			orig: buildLocation(1, 2, []int32{3}, buildLine(1, 2, 3)),
			dest: buildLocation(1, 3, []int32{3}, buildLine(1, 2, 3)),
			want: false,
		},
		{
			name: "with non-equal attribute indices",
			orig: buildLocation(1, 2, []int32{3}, buildLine(1, 2, 3)),
			dest: buildLocation(1, 2, []int32{5}, buildLine(1, 2, 3)),
			want: false,
		},
		{
			name: "with non-equal lines",
			orig: buildLocation(1, 2, []int32{3}, buildLine(4, 5, 6)),
			dest: buildLocation(1, 2, []int32{3}, buildLine(1, 2, 3)),
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

func TestLocationSwitchDictionary(t *testing.T) {
	for _, tt := range []struct {
		name     string
		location Location

		src ProfilesDictionary
		dst ProfilesDictionary

		wantLocation   Location
		wantDictionary ProfilesDictionary
		wantErr        error
	}{
		{
			name:     "with an empty location",
			location: NewLocation(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantLocation:   NewLocation(),
			wantDictionary: NewProfilesDictionary(),
		},
		{
			name: "with an existing mapping",
			location: func() Location {
				l := NewLocation()
				l.SetMappingIndex(1)
				return l
			}(),

			src: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "test")

				d.MappingTable().AppendEmpty()
				m := d.MappingTable().AppendEmpty()
				m.SetFilenameStrindex(1)
				return d
			}(),
			dst: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo")

				d.MappingTable().AppendEmpty()
				d.MappingTable().AppendEmpty()
				return d
			}(),

			wantLocation: func() Location {
				l := NewLocation()
				l.SetMappingIndex(2)
				return l
			}(),
			wantDictionary: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo", "test")

				d.MappingTable().AppendEmpty()
				d.MappingTable().AppendEmpty()
				m := d.MappingTable().AppendEmpty()
				m.SetFilenameStrindex(2)
				return d
			}(),
		},
		{
			name: "with a mapping that cannot be found",
			location: func() Location {
				l := NewLocation()
				l.SetMappingIndex(1)
				return l
			}(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantLocation: func() Location {
				l := NewLocation()
				l.SetMappingIndex(1)
				return l
			}(),
			wantDictionary: NewProfilesDictionary(),
			wantErr:        errors.New("invalid mapping index 1"),
		},
		{
			name: "with an existing attribute",
			location: func() Location {
				l := NewLocation()
				l.AttributeIndices().Append(1)
				return l
			}(),

			src: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "test")

				d.AttributeTable().AppendEmpty()
				a := d.AttributeTable().AppendEmpty()
				a.SetKeyStrindex(1)

				return d
			}(),
			dst: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo")

				d.AttributeTable().AppendEmpty()
				d.AttributeTable().AppendEmpty()
				return d
			}(),

			wantLocation: func() Location {
				l := NewLocation()
				l.AttributeIndices().Append(2)
				return l
			}(),
			wantDictionary: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo", "test")

				d.AttributeTable().AppendEmpty()
				d.AttributeTable().AppendEmpty()
				a := d.AttributeTable().AppendEmpty()
				a.SetKeyStrindex(2)
				return d
			}(),
		},
		{
			name: "with an attribute index that does not match anything",
			location: func() Location {
				l := NewLocation()
				l.AttributeIndices().Append(1)
				return l
			}(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantLocation: func() Location {
				l := NewLocation()
				l.AttributeIndices().Append(1)
				return l
			}(),
			wantDictionary: NewProfilesDictionary(),
			wantErr:        errors.New("invalid attribute index 1"),
		},
		{
			name: "with an existing line",
			location: func() Location {
				l := NewLocation()
				l.Lines().AppendEmpty().SetFunctionIndex(1)
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

			wantLocation: func() Location {
				l := NewLocation()
				l.Lines().AppendEmpty().SetFunctionIndex(2)
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
	} {
		t.Run(tt.name, func(t *testing.T) {
			l := tt.location
			dst := tt.dst
			err := l.switchDictionary(tt.src, dst)

			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, tt.wantErr, err)
			}

			assert.Equal(t, tt.wantLocation, l)
			assert.Equal(t, tt.wantDictionary, dst)
		})
	}
}

func BenchmarkLocationSwitchDictionary(b *testing.B) {
	testutil.SkipMemoryBench(b)

	l := NewLocation()
	l.AttributeIndices().Append(1, 2)

	src := NewProfilesDictionary()
	src.StringTable().Append("", "test")
	src.AttributeTable().AppendEmpty()
	src.AttributeTable().AppendEmpty().SetKeyStrindex(1)
	src.AttributeTable().AppendEmpty().SetKeyStrindex(2)

	b.ReportAllocs()

	for b.Loop() {
		b.StopTimer()
		dst := NewProfilesDictionary()
		dst.StringTable().Append("", "foo")
		dst.AttributeTable().AppendEmpty()
		dst.AttributeTable().AppendEmpty().SetKeyStrindex(1)
		b.StartTimer()

		_ = l.switchDictionary(src, dst)
	}
}

func buildLocation(mapIdx int32, addr uint64, attrIdxs []int32, line Line) Location {
	l := NewLocation()
	l.SetMappingIndex(mapIdx)
	l.SetAddress(addr)
	l.AttributeIndices().FromRaw(attrIdxs)
	line.MoveTo(l.Lines().AppendEmpty())
	return l
}
