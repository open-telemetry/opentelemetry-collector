// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
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
				pipe := cfg.Pipelines[component.NewID("traces")]
				pipe.Processors = append(pipe.Processors, pipe.Processors...)
				return cfg
			},
			expected: fmt.Errorf(`service::pipeline::traces: %w`, errors.New(`references processor "nop" multiple times`)),
		},
		{
			name: "missing-pipeline-receivers",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Pipelines[component.NewID("traces")].Receivers = nil
				return cfg
			},
			expected: fmt.Errorf(`service::pipeline::traces: %w`, errMissingServicePipelineReceivers),
		},
		{
			name: "missing-pipeline-exporters",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Pipelines[component.NewID("traces")].Exporters = nil
				return cfg
			},
			expected: fmt.Errorf(`service::pipeline::traces: %w`, errMissingServicePipelineExporters),
		},
		{
			name: "missing-pipelines",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Pipelines = nil
				return cfg
			},
			expected: errMissingServicePipelines,
		},
		{
			name: "invalid-service-pipeline-type",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Pipelines[component.NewID("wrongtype")] = &PipelineConfig{
					Receivers:  []component.ID{component.NewID("nop")},
					Processors: []component.ID{component.NewID("nop")},
					Exporters:  []component.ID{component.NewID("nop")},
				}
				return cfg
			},
			expected: errors.New(`service::pipeline::wrongtype: unknown datatype "wrongtype"`),
		},
		{
			name: "invalid-telemetry-metric-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Telemetry.Metrics.Level = configtelemetry.LevelBasic
				cfg.Telemetry.Metrics.Address = ""
				return cfg
			},
			expected: nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			cfg := test.cfgFn()
			assert.Equal(t, test.expected, cfg.Validate())
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
				Level:   configtelemetry.LevelNormal,
				Address: ":8080",
			},
		},
		Extensions: []component.ID{component.NewID("nop")},
		Pipelines: map[component.ID]*PipelineConfig{
			component.NewID("traces"): {
				Receivers:  []component.ID{component.NewID("nop")},
				Processors: []component.ID{component.NewID("nop")},
				Exporters:  []component.ID{component.NewID("nop")},
			},
		},
	}
}
