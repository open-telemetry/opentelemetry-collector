// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/pipelines"
	"go.opentelemetry.io/collector/service/telemetry"
)

func TestUnmarshalEmpty(t *testing.T) {
	factories, err := nopFactories()
	require.NoError(t, err)

	_, err = unmarshal(confmap.New(), factories)
	assert.NoError(t, err)
}

func TestUnmarshalEmptyAllSections(t *testing.T) {
	factories, err := nopFactories()
	require.NoError(t, err)

	conf := confmap.NewFromStringMap(map[string]any{
		"receivers":  nil,
		"processors": nil,
		"exporters":  nil,
		"connectors": nil,
		"extensions": nil,
		"service":    nil,
	})
	cfg, err := unmarshal(conf, factories)
	require.NoError(t, err)

	zapProdCfg := zap.NewProductionConfig()
	assert.Equal(t, telemetry.LogsConfig{
		Level:       zapProdCfg.Level.Level(),
		Development: zapProdCfg.Development,
		Encoding:    "console",
		Sampling: &telemetry.LogsSamplingConfig{
			Enabled:    true,
			Tick:       10 * time.Second,
			Initial:    10,
			Thereafter: 100,
		},
		DisableCaller:     zapProdCfg.DisableCaller,
		DisableStacktrace: zapProdCfg.DisableStacktrace,
		OutputPaths:       zapProdCfg.OutputPaths,
		ErrorOutputPaths:  zapProdCfg.ErrorOutputPaths,
		InitialFields:     zapProdCfg.InitialFields,
	}, cfg.Service.Telemetry.Logs)
}

func TestUnmarshalUnknownTopLevel(t *testing.T) {
	factories, err := nopFactories()
	require.NoError(t, err)

	conf := confmap.NewFromStringMap(map[string]any{
		"unknown_section": nil,
	})
	_, err = unmarshal(conf, factories)
	assert.ErrorContains(t, err, "'' has invalid keys: unknown_section")
}

func TestPipelineConfigUnmarshalError(t *testing.T) {
	testCases := []struct {
		// test case name (also file name containing config yaml)
		name string
		conf *confmap.Conf
		// string that the error must contain
		expectError string
	}{
		{
			name: "duplicate-pipeline",
			conf: confmap.NewFromStringMap(map[string]any{
				"traces/ pipe": nil,
				"traces /pipe": nil,
			}),
			expectError: "duplicate name",
		},
		{
			name: "invalid-pipeline-name-after-slash",
			conf: confmap.NewFromStringMap(map[string]any{
				"metrics/": nil,
			}),
			expectError: "in \"metrics/\" id: the part after / should not be empty",
		},
		{
			name: "invalid-pipeline-section",
			conf: confmap.NewFromStringMap(map[string]any{
				"traces": map[string]any{
					"unknown_section": nil,
				},
			}),
			expectError: "'[traces]' has invalid keys: unknown_section",
		},
		{
			name: "invalid-pipeline-sub-config",
			conf: confmap.NewFromStringMap(map[string]any{
				"traces": "string",
			}),
			expectError: "'[traces]' expected a map, got 'string'",
		},
		{
			name: "invalid-pipeline-type",
			conf: confmap.NewFromStringMap(map[string]any{
				"/metrics": nil,
			}),
			expectError: "in \"/metrics\" id: the part before / should not be empty",
		},
		{
			name: "invalid-sequence-value",
			conf: confmap.NewFromStringMap(map[string]any{
				"traces": map[string]any{
					"receivers": map[string]any{
						"nop": map[string]any{
							"some": "config",
						},
					},
				},
			}),
			expectError: "'[traces].receivers': source data must be an array or slice, got map",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			pips := new(pipelines.Config)
			err := tt.conf.Unmarshal(&pips)
			assert.ErrorContains(t, err, tt.expectError)
		})
	}
}

func TestServiceUnmarshalError(t *testing.T) {
	testCases := []struct {
		// test case name (also file name containing config yaml)
		name string
		conf *confmap.Conf
		// string that the error must contain
		expectError string
	}{
		{
			name: "invalid-logs-level",
			conf: confmap.NewFromStringMap(map[string]any{
				"telemetry": map[string]any{
					"logs": map[string]any{
						"level": "UNKNOWN",
					},
				},
			}),
			expectError: "error decoding 'telemetry': decoding failed due to the following error(s):\n\nerror decoding 'logs': decoding failed due to the following error(s):\n\nerror decoding 'level': unrecognized level: \"UNKNOWN\"",
		},
		{
			name: "invalid-metrics-level",
			conf: confmap.NewFromStringMap(map[string]any{
				"telemetry": map[string]any{
					"metrics": map[string]any{
						"level": "unknown",
					},
				},
			}),
			expectError: "error decoding 'telemetry': decoding failed due to the following error(s):\n\nerror decoding 'metrics': decoding failed due to the following error(s):\n\nerror decoding 'level': unknown metrics level \"unknown\"",
		},
		{
			name: "invalid-service-extensions-section",
			conf: confmap.NewFromStringMap(map[string]any{
				"extensions": []any{
					map[string]any{
						"nop": map[string]any{
							"some": "config",
						},
					},
				},
			}),
			expectError: "'extensions[0]' has invalid keys: nop",
		},
		{
			name: "invalid-service-section",
			conf: confmap.NewFromStringMap(map[string]any{
				"unknown_section": "string",
			}),
			expectError: "'' has invalid keys: unknown_section",
		},
		{
			name: "invalid-pipelines-config",
			conf: confmap.NewFromStringMap(map[string]any{
				"pipelines": "string",
			}),
			expectError: "'pipelines' expected a map, got 'string'",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.conf.Unmarshal(&service.Config{})
			require.ErrorContains(t, err, tt.expectError)
		})
	}
}
