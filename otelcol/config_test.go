// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/pipelines"
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
			name: "valid-telemetry-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Service.Telemetry = fakeTelemetryConfig{}
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
			name: "invalid-telemetry-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Service.Telemetry = fakeTelemetryConfig{Invalid: true}
				return cfg
			},
			expected: errors.New("service::telemetry: invalid config"),
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
				pipe := cfg.Service.Pipelines[pipeline.NewID(pipeline.SignalTraces)]
				pipe.Receivers = append(pipe.Receivers, component.MustNewIDWithName("nop", "2"))
				return cfg
			},
			expected: errors.New(`service::pipelines::traces: references receiver "nop/2" which is not configured`),
		},
		{
			name: "invalid-processor-reference",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[pipeline.NewID(pipeline.SignalTraces)]
				pipe.Processors = append(pipe.Processors, component.MustNewIDWithName("nop", "2"))
				return cfg
			},
			expected: errors.New(`service::pipelines::traces: references processor "nop/2" which is not configured`),
		},
		{
			name: "invalid-exporter-reference",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[pipeline.NewID(pipeline.SignalTraces)]
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
				pipe := cfg.Service.Pipelines[pipeline.NewID(pipeline.SignalTraces)]
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
				pipe := cfg.Service.Pipelines[pipeline.NewID(pipeline.SignalTraces)]
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
				pipe := cfg.Service.Pipelines[pipeline.NewID(pipeline.SignalTraces)]
				pipe.Receivers = append(pipe.Receivers, component.MustNewIDWithName("nop", "conn2"))
				return cfg
			},
			expected: errors.New(`service::pipelines::traces: references receiver "nop/conn2" which is not configured`),
		},
		{
			name: "invalid-connector-reference-as-receiver",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[pipeline.NewID(pipeline.SignalTraces)]
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
			expected: fmt.Errorf(`service::pipelines: %w`, errors.New(`service must have at least one pipeline`)),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfgFn()
			err := xconfmap.Validate(cfg)
			if tt.expected != nil {
				require.EqualError(t, err, tt.expected.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNoPipelinesFeatureGate(t *testing.T) {
	cfg := generateConfig()
	cfg.Receivers = nil
	cfg.Exporters = nil
	cfg.Service.Pipelines = pipelines.Config{}

	require.Error(t, xconfmap.Validate(cfg))

	gate := pipelines.AllowNoPipelines
	require.NoError(t, featuregate.GlobalRegistry().Set(gate.ID(), true))
	defer func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(gate.ID(), false))
	}()

	require.NoError(t, xconfmap.Validate(cfg))
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
			Telemetry:  fakeTelemetryConfig{},
			Extensions: []component.ID{component.MustNewID("nop")},
			Pipelines: pipelines.Config{
				pipeline.NewID(pipeline.SignalTraces): {
					Receivers:  []component.ID{component.MustNewID("nop")},
					Processors: []component.ID{component.MustNewID("nop")},
					Exporters:  []component.ID{component.MustNewID("nop")},
				},
			},
		},
	}
}

type fakeTelemetryConfig struct {
	Invalid bool `mapstructure:"invalid"`
}

func (cfg fakeTelemetryConfig) Validate() error {
	if cfg.Invalid {
		return errors.New("invalid config")
	}
	return nil
}
