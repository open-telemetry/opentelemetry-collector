// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStackEqual(t *testing.T) {
	for _, tt := range []struct {
		name string
		orig Stack
		dest Stack
		want bool
	}{
		{
			name: "with empty stacks",
			orig: NewStack(),
			dest: NewStack(),
			want: true,
		},
		{
			name: "with non-empty equal stacks",
			orig: func() Stack {
				s := NewStack()
				s.LocationIndices().Append(1)
				return s
			}(),
			dest: func() Stack {
				s := NewStack()
				s.LocationIndices().Append(1)
				return s
			}(),
			want: true,
		},
		{
			name: "with different location indices lengths",
			orig: func() Stack {
				s := NewStack()
				s.LocationIndices().Append(1)
				return s
			}(),
			dest: NewStack(),
			want: false,
		},
		{
			name: "with non-equal location indices",
			orig: func() Stack {
				s := NewStack()
				s.LocationIndices().Append(2)
				return s
			}(),
			dest: func() Stack {
				s := NewStack()
				s.LocationIndices().Append(1)
				return s
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
