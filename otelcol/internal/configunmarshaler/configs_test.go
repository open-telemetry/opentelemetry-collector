// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configunmarshaler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

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

func TestUnmarshal_PerComponentTelemetryLogLevel(t *testing.T) {
	for _, tk := range testKinds {
		t.Run(tk.kind, func(t *testing.T) {
			cfgs := NewConfigs(tk.factories)

			// Config with per-component telemetry.logs.level override
			conf := confmap.NewFromStringMap(map[string]any{
				"nop": map[string]any{
					"telemetry": map[string]any{
						"logs": map[string]any{
							"level": "debug",
						},
					},
				},
				"nop/my" + tk.kind: map[string]any{
					"telemetry": map[string]any{
						"logs": map[string]any{
							"level": "warn",
						},
					},
				},
				// This one has no telemetry override
				"nop/note": nil,
			})

			require.NoError(t, cfgs.Unmarshal(conf))

			// Verify telemetry overrides were extracted
			overrides := cfgs.TelemetryOverrides()
			require.Len(t, overrides, 2)

			debugLevel := zapcore.DebugLevel
			warnLevel := zapcore.WarnLevel

			assert.Equal(t, &debugLevel, overrides[component.NewID(nopType)].LogLevel)
			assert.Equal(t, &warnLevel, overrides[component.NewIDWithName(nopType, "my"+tk.kind)].LogLevel)

			// Verify the component with no telemetry override is not in the map
			_, hasNote := overrides[component.NewIDWithName(nopType, "note")]
			assert.False(t, hasNote)

			// Verify all component configs were still unmarshalled correctly
			cfgsMap := cfgs.Configs()
			assert.Len(t, cfgsMap, 3)
		})
	}
}

func TestUnmarshal_PerComponentTelemetryStrippedFromFactoryConfig(t *testing.T) {
	for _, tk := range testKinds {
		t.Run(tk.kind, func(t *testing.T) {
			cfgs := NewConfigs(tk.factories)

			// Config with telemetry AND some other field
			conf := confmap.NewFromStringMap(map[string]any{
				"nop": map[string]any{
					"telemetry": map[string]any{
						"logs": map[string]any{
							"level": "info",
						},
					},
				},
			})

			require.NoError(t, cfgs.Unmarshal(conf))

			// The factory's config should be created without errors.
			// If "telemetry" had not been stripped, the default nop config
			// would have likely caused an "unknown field" error during
			// unmarshalling. Since we stripped it, this should work fine.
			cfgsMap := cfgs.Configs()
			require.Len(t, cfgsMap, 1)
			assert.Equal(t, tk.factories[nopType].CreateDefaultConfig(), cfgsMap[component.NewID(nopType)])
		})
	}
}

func TestUnmarshal_PerComponentTelemetryInvalidLogLevel(t *testing.T) {
	cfgs := NewConfigs(testKinds[0].factories) // receivers

	// Invalid log level should be silently ignored (no override extracted)
	conf := confmap.NewFromStringMap(map[string]any{
		"nop": map[string]any{
			"telemetry": map[string]any{
				"logs": map[string]any{
					"level": "invalid-level",
				},
			},
		},
	})

	require.NoError(t, cfgs.Unmarshal(conf))

	overrides := cfgs.TelemetryOverrides()
	assert.Len(t, overrides, 1) // telemetry key was still detected
	assert.Nil(t, overrides[component.NewID(nopType)].LogLevel)
}

func TestUnmarshal_NoTelemetryNoOverride(t *testing.T) {
	cfgs := NewConfigs(testKinds[0].factories) // receivers

	// Config without any telemetry
	conf := confmap.NewFromStringMap(map[string]any{
		"nop": nil,
	})

	require.NoError(t, cfgs.Unmarshal(conf))

	overrides := cfgs.TelemetryOverrides()
	assert.Empty(t, overrides)
}
