// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/pipelines"
	"go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"
)

func TestConfigValidate(t *testing.T) {
	testCases := []struct {
		name     string // test case name (also file name containing config yaml)
		cfgFn    func() *Config
		expected error
	}{
		{
			name:     "valid",
			cfgFn:    generateConfig,
			expected: nil,
		},
		{
			name: "custom-service-telemetrySettings-encoding",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Telemetry.Logs.Encoding = "json"
				return cfg
			},
			expected: nil,
		},
		{
			name: "duplicate-processor-reference",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Pipelines[pipeline.NewID(pipeline.SignalTraces)]
				pipe.Processors = append(pipe.Processors, pipe.Processors...)
				return cfg
			},
			expected: errors.New(`references processor "nop" multiple times`),
		},
		{
			name: "invalid-telemetry-metric-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Telemetry.Metrics.Level = configtelemetry.LevelBasic
				cfg.Telemetry.Metrics.Readers = nil
				return cfg
			},
			expected: errors.New("collector telemetry metrics reader should exist when metric level is not none"),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfgFn()
			err := xconfmap.Validate(cfg)
			if tt.expected != nil {
				assert.ErrorContains(t, err, tt.expected.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfmapMarshalConfig(t *testing.T) {
	telFactory := otelconftelemetry.NewFactory()
	defaultTelConfig := *telFactory.CreateDefaultConfig().(*otelconftelemetry.Config)
	conf := confmap.New()

	require.NoError(t, conf.Marshal(Config{
		Telemetry: defaultTelConfig,
	}))
	assert.Equal(t, map[string]any{
		"pipelines": map[string]any(nil),
		"telemetry": map[string]any{
			"logs": map[string]any{
				"encoding":           "console",
				"level":              "info",
				"error_output_paths": []any{"stderr"},
				"output_paths":       []any{"stderr"},
				"sampling": map[string]any{
					"enabled":    true,
					"initial":    10,
					"thereafter": 100,
					"tick":       10 * time.Second,
				},
			},
			"metrics": map[string]any{
				"level": "Normal",
				"readers": []any{
					map[string]any{
						"pull": map[string]any{
							"exporter": map[string]any{
								"prometheus": map[string]any{
									"host": "localhost",
									"port": 8888,
									"with_resource_constant_labels": map[string]any{
										"included": []any{},
									},
									"without_scope_info":  true,
									"without_type_suffix": true,
									"without_units":       true,
								},
							},
						},
					},
				},
			},
		},
	}, conf.ToStringMap())
}

func generateConfig() *Config {
	return &Config{
		Telemetry: otelconftelemetry.Config{
			Logs: otelconftelemetry.LogsConfig{
				Level:             zapcore.DebugLevel,
				Development:       true,
				Encoding:          "console",
				DisableCaller:     true,
				DisableStacktrace: true,
				OutputPaths:       []string{"stderr", "./output-logs"},
				ErrorOutputPaths:  []string{"stderr", "./error-output-logs"},
				InitialFields:     map[string]any{"fieldKey": "filed-value"},
			},
			Metrics: otelconftelemetry.MetricsConfig{
				Level: configtelemetry.LevelNormal,
				MeterProvider: config.MeterProvider{
					Readers: []config.MetricReader{
						{
							Pull: &config.PullMetricReader{Exporter: config.PullMetricExporter{Prometheus: &config.Prometheus{
								Host: newPtr("localhost"),
								Port: newPtr(8080),
							}}},
						},
					},
				},
			},
		},
		Extensions: extensions.Config{component.MustNewID("nop")},
		Pipelines: pipelines.Config{
			pipeline.NewID(pipeline.SignalTraces): {
				Receivers:  []component.ID{component.MustNewID("nop")},
				Processors: []component.ID{component.MustNewID("nop")},
				Exporters:  []component.ID{component.MustNewID("nop")},
			},
		},
	}
}
