// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestStackSwitchDictionary(t *testing.T) {
	for _, tt := range []struct {
		name  string
		stack Stack

		src ProfilesDictionary
		dst ProfilesDictionary

		wantStack      Stack
		wantDictionary ProfilesDictionary
		wantErr        error
	}{
		{
			name:  "with an empty stack",
			stack: NewStack(),

			src: NewProfilesDictionary(),
			dst: NewProfilesDictionary(),

			wantStack:      NewStack(),
			wantDictionary: NewProfilesDictionary(),
		},
		{
			name: "with an existing location",
			stack: func() Stack {
				s := NewStack()
				s.LocationIndices().Append(0)
				return s
			}(),

			src: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				loc := d.LocationTable().AppendEmpty()
				loc.SetAddress(42)
				return d
			}(),
			dst: NewProfilesDictionary(),

			wantStack: func() Stack {
				s := NewStack()
				s.LocationIndices().Append(0)
				return s
			}(),
			wantDictionary: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				loc := d.LocationTable().AppendEmpty()
				loc.SetAddress(42)
				return d
			}(),
		},
		{
			name: "with an existing location that needs a new indice",
			stack: func() Stack {
				s := NewStack()
				s.LocationIndices().Append(0)
				return s
			}(),

			src: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				loc := d.LocationTable().AppendEmpty()
				loc.SetAddress(42)
				return d
			}(),
			dst: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				loc := d.LocationTable().AppendEmpty()
				loc.SetAddress(2)
				return d
			}(),

			wantStack: func() Stack {
				s := NewStack()
				s.LocationIndices().Append(1)
				return s
			}(),
			wantDictionary: func() ProfilesDictionary {
				d := NewProfilesDictionary()
				loc := d.LocationTable().AppendEmpty()
				loc.SetAddress(2)
				loc = d.LocationTable().AppendEmpty()
				loc.SetAddress(42)
				return d
			}(),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stack := tt.stack
			dst := tt.dst
			err := stack.switchDictionary(tt.src, dst)

			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, tt.wantErr, err)
			}

			assert.Equal(t, tt.wantStack, stack)
			assert.Equal(t, tt.wantDictionary, dst)
		})
	}
}
