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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
)

func TestReceiversUnmarshal(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	recvs := NewReceivers(factories.Receivers)
	conf := confmap.NewFromStringMap(map[string]interface{}{
		"nop":            nil,
		"nop/myreceiver": nil,
	})
	require.NoError(t, recvs.Unmarshal(conf))

	cfgWithName := factories.Receivers["nop"].CreateDefaultConfig()
	cfgWithName.SetIDName("myreceiver")
	assert.Equal(t, map[component.ID]component.ReceiverConfig{
		component.NewID("nop"):                       factories.Receivers["nop"].CreateDefaultConfig(),
		component.NewIDWithName("nop", "myreceiver"): cfgWithName,
	}, recvs.GetReceivers())
}

func TestReceiversUnmarshalError(t *testing.T) {
	var testCases = []struct {
		name string
		conf *confmap.Conf
		// string that the error must contain
		expectedError string
	}{
		{
			name: "invalid-receiver-type",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nop":     nil,
				"/custom": nil,
			}),
			expectedError: "the part before / should not be empty",
		},
		{
			name: "invalid-receiver-name-after-slash",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nop":  nil,
				"nop/": nil,
			}),
			expectedError: "the part after / should not be empty",
		},
		{
			name: "unknown-receiver-type",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nosuchreceiver": nil,
			}),
			expectedError: "unknown receivers type: \"nosuchreceiver\"",
		},
		{
			name: "duplicate-receiver",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nop /exp ": nil,
				" nop/ exp": nil,
			}),
			expectedError: "duplicate name",
		},
		{
			name: "invalid-receiver-section",
			conf: confmap.NewFromStringMap(map[string]interface{}{
				"nop": map[string]interface{}{
					"unknown_section": "receiver",
				},
			}),
			expectedError: "error reading receivers configuration for \"nop\"",
		},
		{
			name: "invalid-receiver-sub-config",
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
			recvs := NewReceivers(factories.Receivers)
			err = recvs.Unmarshal(tt.conf)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}
