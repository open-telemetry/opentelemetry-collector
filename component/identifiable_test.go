// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalText(t *testing.T) {
	id := NewIDWithName(MustNewType("test"), "name")
	got, err := id.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, id.String(), string(got))
}

func TestUnmarshalText(t *testing.T) {
	validType := MustNewType("valid_type")
	var testCases = []struct {
		name        string
		expectedErr bool
		expectedID  ID
	}{
		{
			name:       "valid_type",
			expectedID: ID{typeVal: validType, nameVal: ""},
		},
		{
			name:       "valid_type/valid_name",
			expectedID: ID{typeVal: validType, nameVal: "valid_name"},
		},
		{
			name:       "   valid_type   /   valid_name  ",
			expectedID: ID{typeVal: validType, nameVal: "valid_name"},
		},
		{
			name:       "valid_type/中文好",
			expectedID: ID{typeVal: validType, nameVal: "中文好"},
		},
		{
			name:       "valid_type/name-with-dashes",
			expectedID: ID{typeVal: validType, nameVal: "name-with-dashes"},
		},
		// issue 10816
		{
			name:       "valid_type/Linux-Messages-File_01J49HCH3SWFXRVASWFZFRT3J2__processor0__logs",
			expectedID: ID{typeVal: validType, nameVal: "Linux-Messages-File_01J49HCH3SWFXRVASWFZFRT3J2__processor0__logs"},
		},
		{
			name:       "valid_type/1",
			expectedID: ID{typeVal: validType, nameVal: "1"},
		},
		{
			name:        "/valid_name",
			expectedErr: true,
		},
		{
			name:        "     /valid_name",
			expectedErr: true,
		},
		{
			name:        "valid_type/",
			expectedErr: true,
		},
		{
			name:        "valid_type/      ",
			expectedErr: true,
		},
		{
			name:        "      ",
			expectedErr: true,
		},
		{
			name:        "valid_type/invalid name",
			expectedErr: true,
		},
		{
			name:        "valid_type/" + strings.Repeat("a", 1025),
			expectedErr: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			id := ID{}
			err := id.UnmarshalText([]byte(tt.name))
			if tt.expectedErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedID, id)
			assert.Equal(t, tt.expectedID.Type(), id.Type())
			assert.Equal(t, tt.expectedID.Name(), id.Name())
			assert.Equal(t, tt.expectedID.String(), id.String())
		})
	}
}
