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

package configauth

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarshalJSON(t *testing.T) {
	for _, tt := range []struct {
		name     string
		ac       *authC
		expected string
	}{
		{
			name: "simple",
			ac: &authC{
				delegate: map[interface{}]interface{}{
					"subject": "username",
					"tenant":  "acme",
				},
			},
			expected: `{"subject":"username","tenant":"acme"}`,
		},
		{
			name: "groups",
			ac: &authC{
				delegate: map[interface{}]interface{}{
					"subject":    "username",
					"membership": []string{"dev", "ops"},
				},
			},
			expected: `{"membership":["dev","ops"],"subject":"username"}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			json, err := tt.ac.MarshalJSON()
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, string(json))
		})
	}
}

func TestUnmarshalJSON(t *testing.T) {
	for _, tt := range []struct {
		name     string
		json     string
		expected *authC
	}{
		{
			name: "simple",
			json: `{"subject":"username","tenant":"acme"}`,
			expected: &authC{
				delegate: map[interface{}]interface{}{
					"subject": "username",
					"tenant":  "acme",
				},
			},
		},
		{
			name: "groups",
			json: `{"membership":["dev","ops"],"subject":"username"}`,
			expected: &authC{
				delegate: map[interface{}]interface{}{
					"subject":    "username",
					"membership": []interface{}{"dev", "ops"},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ac := &authC{}
			err := json.Unmarshal([]byte(tt.json), ac)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, ac)
		})
	}
}
