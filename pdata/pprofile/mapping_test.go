// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
			orig: buildMapping(1, 2, 3, 4, []int32{1, 2}, true, true, true, true),
			dest: buildMapping(1, 2, 3, 4, []int32{1, 2}, true, true, true, true),
			want: true,
		},
		{
			name: "with different MemoryStart",
			orig: buildMapping(1, 2, 3, 4, []int32{1, 2}, true, true, true, true),
			dest: buildMapping(2, 2, 3, 4, []int32{1, 2}, true, true, true, true),
			want: false,
		},
		{
			name: "with different MemoryLimit",
			orig: buildMapping(1, 2, 3, 4, []int32{1, 2}, true, true, true, true),
			dest: buildMapping(1, 3, 3, 4, []int32{1, 2}, true, true, true, true),
			want: false,
		},
		{
			name: "with different FileOffset",
			orig: buildMapping(1, 2, 3, 4, []int32{1, 2}, true, true, true, true),
			dest: buildMapping(1, 2, 4, 4, []int32{1, 2}, true, true, true, true),
			want: false,
		},
		{
			name: "with different FilenameStrindex",
			orig: buildMapping(1, 2, 3, 4, []int32{1, 2}, true, true, true, true),
			dest: buildMapping(1, 2, 3, 5, []int32{1, 2}, true, true, true, true),
			want: false,
		},
		{
			name: "with different AttributeIndices",
			orig: buildMapping(1, 2, 3, 4, []int32{1, 2}, true, true, true, true),
			dest: buildMapping(1, 2, 3, 4, []int32{1, 3}, true, true, true, true),
			want: false,
		},
		{
			name: "with different HasFunctions",
			orig: buildMapping(1, 2, 3, 4, []int32{1, 2}, true, true, true, true),
			dest: buildMapping(1, 2, 3, 4, []int32{1, 2}, false, true, true, true),
			want: false,
		},
		{
			name: "with different HasFilenames",
			orig: buildMapping(1, 2, 3, 4, []int32{1, 2}, true, true, true, true),
			dest: buildMapping(1, 2, 3, 4, []int32{1, 2}, true, false, true, true),
			want: false,
		},
		{
			name: "with different HasLineNumbers",
			orig: buildMapping(1, 2, 3, 4, []int32{1, 2}, true, true, true, true),
			dest: buildMapping(1, 2, 3, 4, []int32{1, 2}, true, true, false, true),
			want: false,
		},
		{
			name: "with different HasInlineFrames",
			orig: buildMapping(1, 2, 3, 4, []int32{1, 2}, true, true, true, true),
			dest: buildMapping(1, 2, 3, 4, []int32{1, 2}, true, true, true, false),
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

func buildMapping(memStart, memLimit, fileOffset uint64, filenameIdx int32, attrIdxs []int32, hasFn, hasFilenames, hasLnNumber, hasInlFrames bool) Mapping {
	m := NewMapping()
	m.SetMemoryStart(memStart)
	m.SetMemoryLimit(memLimit)
	m.SetFileOffset(fileOffset)
	m.SetFilenameStrindex(filenameIdx)
	m.AttributeIndices().FromRaw(attrIdxs)
	m.SetHasFunctions(hasFn)
	m.SetHasFilenames(hasFilenames)
	m.SetHasLineNumbers(hasLnNumber)
	m.SetHasInlineFrames(hasInlFrames)
	return m
}
