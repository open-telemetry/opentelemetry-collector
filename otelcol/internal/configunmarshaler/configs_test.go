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
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

var testKinds = []struct {
	kind      string
	factories map[component.Type]component.Factory
}{
	{
		kind: "receiver",
		factories: map[component.Type]component.Factory{
			"nop": receivertest.NewNopFactory(),
		},
	},
	{
		kind: "processor",
		factories: map[component.Type]component.Factory{
			"nop": processortest.NewNopFactory(),
		},
	},
	{
		kind: "exporter",
		factories: map[component.Type]component.Factory{
			"nop": exportertest.NewNopFactory(),
		},
	},
	{
		kind: "connector",
		factories: map[component.Type]component.Factory{
			"nop": connectortest.NewNopFactory(),
		},
	},
	{
		kind: "extension",
		factories: map[component.Type]component.Factory{
			"nop": extensiontest.NewNopFactory(),
		},
	},
}

func TestUnmarshal(t *testing.T) {
	for _, tk := range testKinds {
		t.Run(tk.kind, func(t *testing.T) {
			cfgs := NewConfigs(tk.factories)
			conf := confmap.NewFromStringMap(map[string]any{
				"nop":              nil,
				"nop/my" + tk.kind: nil,
			})
			require.NoError(t, cfgs.Unmarshal(conf))

			assert.Equal(t, map[component.ID]component.Config{
				component.NewID("nop"):                       tk.factories["nop"].CreateDefaultConfig(),
				component.NewIDWithName("nop", "my"+tk.kind): tk.factories["nop"].CreateDefaultConfig(),
			}, cfgs.Configs())
		})
	}
}

func TestUnmarshalError(t *testing.T) {
	for _, tk := range testKinds {
		t.Run(tk.kind, func(t *testing.T) {
			var testCases = []struct {
				name string
				conf *confmap.Conf
				// string that the error must contain
				expectedError string
			}{
				{
					name: "invalid-type",
					conf: confmap.NewFromStringMap(map[string]any{
						"nop":     nil,
						"/custom": nil,
					}),
					expectedError: "the part before / should not be empty",
				},
				{
					name: "invalid-name-after-slash",
					conf: confmap.NewFromStringMap(map[string]any{
						"nop":  nil,
						"nop/": nil,
					}),
					expectedError: "the part after / should not be empty",
				},
				{
					name: "unknown-type",
					conf: confmap.NewFromStringMap(map[string]any{
						"nosuch" + tk.kind: nil,
					}),
					expectedError: "unknown type: \"nosuch" + tk.kind + "\"",
				},
				{
					name: "duplicate",
					conf: confmap.NewFromStringMap(map[string]any{
						"nop /my" + tk.kind + " ": nil,
						" nop/ my" + tk.kind:      nil,
					}),
					expectedError: "duplicate name",
				},
				{
					name: "invalid-section",
					conf: confmap.NewFromStringMap(map[string]any{
						"nop": map[string]any{
							"unknown_section": tk.kind,
						},
					}),
					expectedError: "error reading configuration for \"nop\"",
				},
				{
					name: "invalid-sub-config",
					conf: confmap.NewFromStringMap(map[string]any{
						"nop": "tests",
					}),
					expectedError: "'[nop]' expected a map, got 'string'",
				},
			}

			for _, tt := range testCases {
				t.Run(tt.name, func(t *testing.T) {
					cfgs := NewConfigs(tk.factories)
					err := cfgs.Unmarshal(tt.conf)
					require.Error(t, err)
					assert.Contains(t, err.Error(), tt.expectedError)
				})
			}
		})
	}
}
