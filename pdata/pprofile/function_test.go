// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

func buildFunction(name, sName, fileName int32, startLine int64) Function {
	f := NewFunction()
	f.SetNameStrindex(name)
	f.SetSystemNameStrindex(sName)
	f.SetFilenameStrindex(fileName)
	f.SetStartLine(startLine)
	return f
}
