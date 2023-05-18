// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarshalText(t *testing.T) {
	id := NewIDWithName("test", "name")
	got, err := id.MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, id.String(), string(got))
}

func TestUnmarshalText(t *testing.T) {
	var testCases = []struct {
		idStr       string
		expectedErr bool
		expectedID  ID
	}{
		{
			idStr:      "valid_type",
			expectedID: ID{typeVal: "valid_type", nameVal: ""},
		},
		{
			idStr:      "valid_type/valid_name",
			expectedID: ID{typeVal: "valid_type", nameVal: "valid_name"},
		},
		{
			idStr:      "   valid_type   /   valid_name  ",
			expectedID: ID{typeVal: "valid_type", nameVal: "valid_name"},
		},
		{
			idStr:       "/valid_name",
			expectedErr: true,
		},
		{
			idStr:       "     /valid_name",
			expectedErr: true,
		},
		{
			idStr:       "valid_type/",
			expectedErr: true,
		},
		{
			idStr:       "valid_type/      ",
			expectedErr: true,
		},
		{
			idStr:       "      ",
			expectedErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.idStr, func(t *testing.T) {
			id := ID{}
			err := id.UnmarshalText([]byte(test.idStr))
			if test.expectedErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, test.expectedID, id)
			assert.Equal(t, test.expectedID.Type(), id.Type())
			assert.Equal(t, test.expectedID.Name(), id.Name())
			assert.Equal(t, test.expectedID.String(), id.String())
		})
	}
}
