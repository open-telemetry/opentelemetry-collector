// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIDFromString(t *testing.T) {
	var testCases = []struct {
		idStr       string
		expectedErr bool
		expectedID  ComponentID
	}{
		{
			idStr:      "valid_type",
			expectedID: ComponentID{typeVal: "valid_type", nameVal: ""},
		},
		{
			idStr:      "valid_type/valid_name",
			expectedID: ComponentID{typeVal: "valid_type", nameVal: "valid_name"},
		},
		{
			idStr:      "   valid_type   /   valid_name  ",
			expectedID: ComponentID{typeVal: "valid_type", nameVal: "valid_name"},
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
			id, err := NewIDFromString(test.idStr)
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
