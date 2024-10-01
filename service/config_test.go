// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/contrib/config"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/pipelines"
	"go.opentelemetry.io/collector/service/telemetry"
)

func TestConfigValidate(t *testing.T) {
	var testCases = []struct {
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
				pipe := cfg.Pipelines[pipeline.MustNewID("traces")]
				pipe.Processors = append(pipe.Processors, pipe.Processors...)
				return cfg
			},
			expected: fmt.Errorf(`service::pipelines config validation failed: %w`, fmt.Errorf(`pipeline "traces": %w`, errors.New(`references processor "nop" multiple times`))),
		},
		{
			name: "invalid-service-pipeline-type",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Pipelines[pipeline.MustNewID("wrongtype")] = &pipelines.PipelineConfig{
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				}
				return cfg
			},
			expected: fmt.Errorf(`service::pipelines config validation failed: %w`, errors.New(`pipeline "wrongtype": unknown signal "wrongtype"`)),
		},
		{
			name: "invalid-telemetry-metric-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Telemetry.Metrics.Level = configtelemetry.LevelBasic
				cfg.Telemetry.Metrics.Readers = nil
				return cfg
			},
			expected: nil,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfgFn()
			assert.Equal(t, tt.expected, cfg.Validate())
		})
	}
}

func generateConfig() *Config {
	return &Config{
		Telemetry: telemetry.Config{
			Logs: telemetry.LogsConfig{
				Level:             zapcore.DebugLevel,
				Development:       true,
				Encoding:          "console",
				DisableCaller:     true,
				DisableStacktrace: true,
				OutputPaths:       []string{"stderr", "./output-logs"},
				ErrorOutputPaths:  []string{"stderr", "./error-output-logs"},
				InitialFields:     map[string]any{"fieldKey": "filed-value"},
			},
			Metrics: telemetry.MetricsConfig{
				Level: configtelemetry.LevelNormal,
				Readers: []config.MetricReader{{
					Pull: &config.PullMetricReader{Exporter: config.MetricExporter{Prometheus: &config.Prometheus{
						Host: newPtr("localhost"),
						Port: newPtr(8080),
					}}}},
				},
			},
		},
		Extensions: extensions.Config{component.MustNewID("nop")},
		Pipelines: pipelines.Config{
			pipeline.MustNewID("traces"): {
				Receivers:  []component.ID{component.MustNewID("nop")},
				Processors: []component.ID{component.MustNewID("nop")},
				Exporters:  []component.ID{component.MustNewID("nop")},
			},
		},
	}
}
