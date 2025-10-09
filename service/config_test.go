// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/pipelines"
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
			name: "valid-telemetry-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Telemetry = fakeTelemetryConfig{Invalid: false}
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
			name: "invalid-telemetry-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Telemetry = fakeTelemetryConfig{Invalid: true}
				return cfg
			},
			expected: errors.New("telemetry: invalid config"),
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
	conf := confmap.New()

	require.NoError(t, conf.Marshal(Config{
		Telemetry: fakeTelemetryConfig{},
	}))
	assert.Equal(t, map[string]any{
		"pipelines": map[string]any(nil),
		"telemetry": map[string]any{"invalid": false},
	}, conf.ToStringMap())
}

func generateConfig() *Config {
	return &Config{
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

type fakeTelemetryConfig struct {
	Invalid bool `mapstructure:"invalid"`
}

func (cfg fakeTelemetryConfig) Validate() error {
	if cfg.Invalid {
		return errors.New("invalid config")
	}
	return nil
}
