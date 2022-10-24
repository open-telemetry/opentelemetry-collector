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

package configunmarshaler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/confmap"
)

func TestExtensionsUnmarshal(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	exts := NewExtensions(factories.Extensions)
	conf := confmap.NewFromStringMap(map[string]interface{}{
		"nop":             nil,
		"nop/myextension": nil,
	})
	require.NoError(t, exts.Unmarshal(conf))

	cfgWithName := factories.Extensions["nop"].CreateDefaultConfig()
	cfgWithName.SetIDName("myextension")
	assert.Equal(t, map[config.ComponentID]config.Extension{
		config.NewComponentID("nop"):                        factories.Extensions["nop"].CreateDefaultConfig(),
		config.NewComponentIDWithName("nop", "myextension"): cfgWithName,
	}, exts.GetExtensions())
}

func TestExtensionsUnmarshalError(t *testing.T) {
	var testCases = []struct {
		name string
		conf *confmap.Conf
		// string that the error must contain
		expectedError string
	}{
		{
			name: "invalid-extension-type",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nop":     nil,
				"/custom": nil,
			}),
			expectedError: "the part before / should not be empty",
		},
		{
			name: "invalid-extension-name-after-slash",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nop":  nil,
				"nop/": nil,
			}),
			expectedError: "the part after / should not be empty",
		},
		{
			name: "unknown-extension-type",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nosuchextension": nil,
			}),
			expectedError: "unknown extensions type: \"nosuchextension\"",
		},
		{
			name: "duplicate-extension",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nop /exp ": nil,
				" nop/ exp": nil,
			}),
			expectedError: "duplicate name",
		},
		{
			name: "invalid-extension-section",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nop": map[string]interface{}{
					"unknown_section": "extension",
				},
			}),
			expectedError: "error reading extensions configuration for \"nop\"",
		},
		{
			name: "invalid-extension-sub-config",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nop": "tests",
			}),
			expectedError: "'[nop]' expected a map, got 'string'",
		},
	}

	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			exts := NewExtensions(factories.Extensions)
			err = exts.Unmarshal(tt.conf)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}
