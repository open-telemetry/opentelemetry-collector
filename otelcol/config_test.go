// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/pipelines"
	"go.opentelemetry.io/collector/service/telemetry"
)

var (
	errInvalidRecvConfig = errors.New("invalid receiver config")
	errInvalidExpConfig  = errors.New("invalid exporter config")
	errInvalidProcConfig = errors.New("invalid processor config")
	errInvalidConnConfig = errors.New("invalid connector config")
	errInvalidExtConfig  = errors.New("invalid extension config")
)

type errConfig struct {
	validateErr error
}

func (c *errConfig) Validate() error {
	return c.validateErr
}

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
				cfg.Service.Telemetry.Logs.Encoding = "json"
				return cfg
			},
			expected: nil,
		},
		{
			name: "empty configuration file",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Receivers = nil
				cfg.Connectors = nil
				cfg.Processors = nil
				cfg.Exporters = nil
				cfg.Extensions = nil
				return cfg
			},
			expected: errEmptyConfigurationFile,
		},
		{
			name: "missing-exporters",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Exporters = nil
				return cfg
			},
			expected: errMissingExporters,
		},
		{
			name: "missing-receivers",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Receivers = nil
				return cfg
			},
			expected: errMissingReceivers,
		},
		{
			name: "invalid-extension-reference",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Service.Extensions = append(cfg.Service.Extensions, component.MustNewIDWithName("nop", "2"))
				return cfg
			},
			expected: errors.New(`service::extensions: references extension "nop/2" which is not configured`),
		},
		{
			name: "invalid-receiver-reference",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[component.MustNewID("traces")]
				pipe.Receivers = append(pipe.Receivers, component.MustNewIDWithName("nop", "2"))
				return cfg
			},
			expected: errors.New(`service::pipelines::traces: references receiver "nop/2" which is not configured`),
		},
		{
			name: "invalid-processor-reference",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[component.MustNewID("traces")]
				pipe.Processors = append(pipe.Processors, component.MustNewIDWithName("nop", "2"))
				return cfg
			},
			expected: errors.New(`service::pipelines::traces: references processor "nop/2" which is not configured`),
		},
		{
			name: "invalid-exporter-reference",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[component.MustNewID("traces")]
				pipe.Exporters = append(pipe.Exporters, component.MustNewIDWithName("nop", "2"))
				return cfg
			},
			expected: errors.New(`service::pipelines::traces: references exporter "nop/2" which is not configured`),
		},
		{
			name: "invalid-receiver-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Receivers[component.MustNewID("nop")] = &errConfig{
					validateErr: errInvalidRecvConfig,
				}
				return cfg
			},
			expected: fmt.Errorf(`receivers::nop: %w`, errInvalidRecvConfig),
		},
		{
			name: "invalid-exporter-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Exporters[component.MustNewID("nop")] = &errConfig{
					validateErr: errInvalidExpConfig,
				}
				return cfg
			},
			expected: fmt.Errorf(`exporters::nop: %w`, errInvalidExpConfig),
		},
		{
			name: "invalid-processor-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Processors[component.MustNewID("nop")] = &errConfig{
					validateErr: errInvalidProcConfig,
				}
				return cfg
			},
			expected: fmt.Errorf(`processors::nop: %w`, errInvalidProcConfig),
		},
		{
			name: "invalid-extension-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Extensions[component.MustNewID("nop")] = &errConfig{
					validateErr: errInvalidExtConfig,
				}
				return cfg
			},
			expected: fmt.Errorf(`extensions::nop: %w`, errInvalidExtConfig),
		},
		{
			name: "invalid-connector-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Connectors[component.MustNewIDWithName("nop", "conn")] = &errConfig{
					validateErr: errInvalidConnConfig,
				}
				return cfg
			},
			expected: fmt.Errorf(`connectors::nop/conn: %w`, errInvalidConnConfig),
		},
		{
			name: "ambiguous-connector-name-as-receiver",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Receivers[component.MustNewID("nop2")] = &errConfig{}
				cfg.Connectors[component.MustNewID("nop2")] = &errConfig{}
				pipe := cfg.Service.Pipelines[component.MustNewID("traces")]
				pipe.Receivers = append(pipe.Receivers, component.MustNewIDWithName("nop", "2"))
				pipe.Exporters = append(pipe.Exporters, component.MustNewIDWithName("nop", "2"))
				return cfg
			},
			expected: errors.New(`connectors::nop2: ambiguous ID: Found both "nop2" receiver and "nop2" connector. Change one of the components' IDs to eliminate ambiguity (e.g. rename "nop2" connector to "nop2/connector")`),
		},
		{
			name: "ambiguous-connector-name-as-exporter",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Exporters[component.MustNewID("nop2")] = &errConfig{}
				cfg.Connectors[component.MustNewID("nop2")] = &errConfig{}
				pipe := cfg.Service.Pipelines[component.MustNewID("traces")]
				pipe.Receivers = append(pipe.Receivers, component.MustNewIDWithName("nop", "2"))
				pipe.Exporters = append(pipe.Exporters, component.MustNewIDWithName("nop", "2"))
				return cfg
			},
			expected: errors.New(`connectors::nop2: ambiguous ID: Found both "nop2" exporter and "nop2" connector. Change one of the components' IDs to eliminate ambiguity (e.g. rename "nop2" connector to "nop2/connector")`),
		},
		{
			name: "invalid-connector-reference-as-receiver",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[component.MustNewID("traces")]
				pipe.Receivers = append(pipe.Receivers, component.MustNewIDWithName("nop", "conn2"))
				return cfg
			},
			expected: errors.New(`service::pipelines::traces: references receiver "nop/conn2" which is not configured`),
		},
		{
			name: "invalid-connector-reference-as-receiver",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[component.MustNewID("traces")]
				pipe.Exporters = append(pipe.Exporters, component.MustNewIDWithName("nop", "conn2"))
				return cfg
			},
			expected: errors.New(`service::pipelines::traces: references exporter "nop/conn2" which is not configured`),
		},
		{
			name: "invalid-service-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Service.Pipelines = nil
				return cfg
			},
			expected: fmt.Errorf(`service::pipelines config validation failed: %w`, errors.New(`service must have at least one pipeline`)),
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
		Receivers: map[component.ID]component.Config{
			component.MustNewID("nop"): &errConfig{},
		},
		Exporters: map[component.ID]component.Config{
			component.MustNewID("nop"): &errConfig{},
		},
		Processors: map[component.ID]component.Config{
			component.MustNewID("nop"): &errConfig{},
		},
		Connectors: map[component.ID]component.Config{
			component.MustNewIDWithName("nop", "conn"): &errConfig{},
		},
		Extensions: map[component.ID]component.Config{
			component.MustNewID("nop"): &errConfig{},
		},
		Service: service.Config{
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
			Extensions: []component.ID{component.MustNewID("nop")},
			Pipelines: pipelines.Config{
				component.MustNewID("traces"): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				},
			},
		},
	}
}
