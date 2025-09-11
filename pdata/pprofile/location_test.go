// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

func buildLocation(mapIdx int32, addr uint64, attrIdxs []int32, line Line) Location {
	l := NewLocation()
	l.SetMappingIndex(mapIdx)
	l.SetAddress(addr)
	l.AttributeIndices().FromRaw(attrIdxs)
	line.MoveTo(l.Line().AppendEmpty())
	return l
}
