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

func TestProcessorsUnmarshal(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	procs := NewProcessors(factories.Processors)
	conf := confmap.NewFromStringMap(map[string]interface{}{
		"nop":             nil,
		"nop/myprocessor": nil,
	})
	require.NoError(t, procs.Unmarshal(conf))

	cfgWithName := factories.Processors["nop"].CreateDefaultConfig()
	cfgWithName.SetIDName("myprocessor")
	assert.Equal(t, map[config.ComponentID]config.Processor{
		config.NewComponentID("nop"):                        factories.Processors["nop"].CreateDefaultConfig(),
		config.NewComponentIDWithName("nop", "myprocessor"): cfgWithName,
	}, procs.procs)
}

func TestProcessorsUnmarshalError(t *testing.T) {
	var testCases = []struct {
		name string
		conf *confmap.Conf
		// string that the error must contain
		expectedError string
	}{
		{
			name: "invalid-processor-type",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nop":     nil,
				"/custom": nil,
			}),
			expectedError: "the part before / should not be empty",
		},
		{
			name: "invalid-processor-name-after-slash",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nop":  nil,
				"nop/": nil,
			}),
			expectedError: "the part after / should not be empty",
		},
		{
			name: "unknown-processor-type",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nosuchprocessor": nil,
			}),
			expectedError: "unknown processors type: \"nosuchprocessor\"",
		},
		{
			name: "duplicate-processor",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nop /exp ": nil,
				" nop/ exp": nil,
			}),
			expectedError: "duplicate name",
		},
		{
			name: "invalid-processor-section",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nop": map[string]interface{}{
					"unknown_section": "processor",
				},
			}),
			expectedError: "error reading processors configuration for \"nop\"",
		},
		{
			name: "invalid-processor-sub-config",
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
			procs := NewProcessors(factories.Processors)
			err = procs.Unmarshal(tt.conf)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}
