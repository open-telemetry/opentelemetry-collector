// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMappingEqual(t *testing.T) {
	for _, tt := range []struct {
		name string
		orig Mapping
		dest Mapping
		want bool
	}{
		{
			name: "empty mappings",
			orig: NewMapping(),
			dest: NewMapping(),
			want: true,
		},
		{
			name: "non-empty identical mappings",
			orig: buildMapping(1, 2, 3, 4, []int32{1, 2}),
			dest: buildMapping(1, 2, 3, 4, []int32{1, 2}),
			want: true,
		},
		{
			name: "with different MemoryStart",
			orig: buildMapping(1, 2, 3, 4, []int32{1, 2}),
			dest: buildMapping(2, 2, 3, 4, []int32{1, 2}),
			want: false,
		},
		{
			name: "with different MemoryLimit",
			orig: buildMapping(1, 2, 3, 4, []int32{1, 2}),
			dest: buildMapping(1, 3, 3, 4, []int32{1, 2}),
			want: false,
		},
		{
			name: "with different FileOffset",
			orig: buildMapping(1, 2, 3, 4, []int32{1, 2}),
			dest: buildMapping(1, 2, 4, 4, []int32{1, 2}),
			want: false,
		},
		{
			name: "with different FilenameStrindex",
			orig: buildMapping(1, 2, 3, 4, []int32{1, 2}),
			dest: buildMapping(1, 2, 3, 5, []int32{1, 2}),
			want: false,
		},
		{
			name: "with different AttributeIndices",
			orig: buildMapping(1, 2, 3, 4, []int32{1, 2}),
			dest: buildMapping(1, 2, 3, 4, []int32{1, 3}),
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

func TestMappingSwitchDictionary(t *testing.T) {
	for _, tt := range []struct {
		name    string
		mapping Mapping

		src ProfilesDictionary
		dst ProfilesDictionary

		wantMapping    Mapping
		wantDictionary ProfilesDictionary
		wantErr        error
	}{
		{
			name:    "with an empty mapping",
			mapping: NewMapping(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantMapping:    NewMapping(),
			wantDictionary: NewProfilesDictionary(),
		},
		{
			name: "with an existing filename",
			mapping: func() Mapping {
				m := NewMapping()
				m.SetFilenameStrindex(1)
				return m
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

			wantMapping: func() Mapping {
				m := NewMapping()
				m.SetFilenameStrindex(2)
				return m
			}(),
			wantDictionary: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				d.StringTable().Append("", "foo", "test")
				return d
			}(),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.mapping
			dst := tt.dst
			err := m.switchDictionary(tt.src, dst)

			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, tt.wantErr, err)
			}

			assert.Equal(t, tt.wantMapping, m)
			assert.Equal(t, tt.wantDictionary, dst)
		})
	}
}

func buildMapping(memStart, memLimit, fileOffset uint64, filenameIdx int32, attrIdxs []int32) Mapping {
	m := NewMapping()
	m.SetMemoryStart(memStart)
	m.SetMemoryLimit(memLimit)
	m.SetFileOffset(fileOffset)
	m.SetFilenameStrindex(filenameIdx)
	m.AttributeIndices().FromRaw(attrIdxs)
	return m
}
