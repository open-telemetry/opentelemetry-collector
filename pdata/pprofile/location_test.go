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
			orig: func() Location {
				l := NewLocation()
				l.SetMappingIndex(1)
				l.SetAddress(2)
				l.AttributeIndices().Append(3)
				l.SetIsFolded(true)
				l.Line().AppendEmpty()
				return l
			}(),
			dest: func() Location {
				l := NewLocation()
				l.SetMappingIndex(1)
				l.SetAddress(2)
				l.AttributeIndices().Append(3)
				l.SetIsFolded(true)
				l.Line().AppendEmpty()
				return l
			}(),
			want: true,
		},
		{
			name: "with non-equal mapping index",
			orig: func() Location {
				l := NewLocation()
				l.SetMappingIndex(1)
				l.SetAddress(2)
				l.AttributeIndices().Append(3)
				l.SetIsFolded(true)
				l.Line().AppendEmpty()
				return l
			}(),
			dest: func() Location {
				l := NewLocation()
				l.SetMappingIndex(2)
				l.SetAddress(2)
				l.AttributeIndices().Append(3)
				l.SetIsFolded(true)
				l.Line().AppendEmpty()
				return l
			}(),
			want: false,
		},
		{
			name: "with non-equal address",
			orig: func() Location {
				l := NewLocation()
				l.SetMappingIndex(1)
				l.SetAddress(3)
				l.AttributeIndices().Append(3)
				l.SetIsFolded(true)
				l.Line().AppendEmpty()
				return l
			}(),
			dest: func() Location {
				l := NewLocation()
				l.SetMappingIndex(1)
				l.SetAddress(2)
				l.AttributeIndices().Append(3)
				l.SetIsFolded(true)
				l.Line().AppendEmpty()
				return l
			}(),
			want: false,
		},
		{
			name: "with non-equal attribute indices",
			orig: func() Location {
				l := NewLocation()
				l.SetMappingIndex(1)
				l.SetAddress(2)
				l.AttributeIndices().Append(2)
				l.SetIsFolded(true)
				l.Line().AppendEmpty()
				return l
			}(),
			dest: func() Location {
				l := NewLocation()
				l.SetMappingIndex(1)
				l.SetAddress(2)
				l.AttributeIndices().Append(3)
				l.SetIsFolded(true)
				l.Line().AppendEmpty()
				return l
			}(),
			want: false,
		},
		{
			name: "with non-equal is folded",
			orig: func() Location {
				l := NewLocation()
				l.SetMappingIndex(1)
				l.SetAddress(2)
				l.AttributeIndices().Append(3)
				l.SetIsFolded(true)
				l.Line().AppendEmpty()
				return l
			}(),
			dest: func() Location {
				l := NewLocation()
				l.SetMappingIndex(1)
				l.SetAddress(2)
				l.AttributeIndices().Append(3)
				l.SetIsFolded(false)
				l.Line().AppendEmpty()
				return l
			}(),
			want: false,
		},
		{
			name: "with non-equal lines",
			orig: func() Location {
				l := NewLocation()
				l.SetMappingIndex(1)
				l.SetAddress(2)
				l.AttributeIndices().Append(3)
				l.SetIsFolded(true)
				l.Line().AppendEmpty()
				return l
			}(),
			dest: func() Location {
				l := NewLocation()
				l.SetMappingIndex(1)
				l.SetAddress(2)
				l.AttributeIndices().Append(3)
				l.SetIsFolded(true)
				li := l.Line().AppendEmpty()
				li.SetLine(3)
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
