// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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

var nopType = component.MustNewType("nop")

var testKinds = []struct {
	kind      string
	factories map[component.Type]component.Factory
}{
	{
		kind: "receiver",
		factories: map[component.Type]component.Factory{
			nopType: receivertest.NewNopFactory(),
		},
	},
	{
		kind: "processor",
		factories: map[component.Type]component.Factory{
			nopType: processortest.NewNopFactory(),
		},
	},
	{
		kind: "exporter",
		factories: map[component.Type]component.Factory{
			nopType: exportertest.NewNopFactory(),
		},
	},
	{
		kind: "connector",
		factories: map[component.Type]component.Factory{
			nopType: connectortest.NewNopFactory(),
		},
	},
	{
		kind: "extension",
		factories: map[component.Type]component.Factory{
			nopType: extensiontest.NewNopFactory(),
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
				component.NewID(nopType):                       tk.factories[nopType].CreateDefaultConfig(),
				component.NewIDWithName(nopType, "my"+tk.kind): tk.factories[nopType].CreateDefaultConfig(),
			}, cfgs.Configs())
		})
	}
}

func TestUnmarshalError(t *testing.T) {
	for _, tk := range testKinds {
		t.Run(tk.kind, func(t *testing.T) {
			testCases := []struct {
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
					expectedError: "unknown type: \"nosuch" + tk.kind + "\" for id: \"nosuch" + tk.kind + "\" (valid values: [nop])",
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
					expectedError: "'[nop]' expected type 'map[string]interface {}', got unconvertible type 'string'",
				},
			}

			for _, tt := range testCases {
				t.Run(tt.name, func(t *testing.T) {
					cfgs := NewConfigs(tk.factories)
					err := cfgs.Unmarshal(tt.conf)
					assert.ErrorContains(t, err, tt.expectedError)
				})
			}
		})
	}
}

func TestUnmarshal_LoggingExporter(t *testing.T) {
	conf := confmap.NewFromStringMap(map[string]any{
		"logging": nil,
	})
	factories := map[component.Type]component.Factory{
		nopType: exportertest.NewNopFactory(),
	}
	cfgs := NewConfigs(factories)
	err := cfgs.Unmarshal(conf)
	assert.ErrorContains(t, err, "the logging exporter has been deprecated, use the debug exporter instead")
}
