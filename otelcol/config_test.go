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
				cfg.Service.Extensions = append(cfg.Service.Extensions, component.NewIDWithName("nop", "2"))
				return cfg
			},
			expected: errors.New(`service::extensions: references extension "nop/2" which is not configured`),
		},
		{
			name: "invalid-receiver-reference",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[component.NewID("traces")]
				pipe.Receivers = append(pipe.Receivers, component.NewIDWithName("nop", "2"))
				return cfg
			},
			expected: errors.New(`service::pipeline::traces: references receiver "nop/2" which is not configured`),
		},
		{
			name: "invalid-processor-reference",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[component.NewID("traces")]
				pipe.Processors = append(pipe.Processors, component.NewIDWithName("nop", "2"))
				return cfg
			},
			expected: errors.New(`service::pipeline::traces: references processor "nop/2" which is not configured`),
		},
		{
			name: "invalid-exporter-reference",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[component.NewID("traces")]
				pipe.Exporters = append(pipe.Exporters, component.NewIDWithName("nop", "2"))
				return cfg
			},
			expected: errors.New(`service::pipeline::traces: references exporter "nop/2" which is not configured`),
		},
		{
			name: "invalid-receiver-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Receivers[component.NewID("nop")] = &errConfig{
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
				cfg.Exporters[component.NewID("nop")] = &errConfig{
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
				cfg.Processors[component.NewID("nop")] = &errConfig{
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
				cfg.Extensions[component.NewID("nop")] = &errConfig{
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
				cfg.Connectors[component.NewIDWithName("nop", "conn")] = &errConfig{
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
				cfg.Receivers[component.NewID("nop/2")] = &errConfig{}
				cfg.Connectors[component.NewID("nop/2")] = &errConfig{}
				pipe := cfg.Service.Pipelines[component.NewID("traces")]
				pipe.Receivers = append(pipe.Receivers, component.NewIDWithName("nop", "2"))
				pipe.Exporters = append(pipe.Exporters, component.NewIDWithName("nop", "2"))
				return cfg
			},
			expected: errors.New(`connectors::nop/2: there's already a receiver named "nop/2"`),
		},
		{
			name: "ambiguous-connector-name-as-exporter",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Exporters[component.NewID("nop/2")] = &errConfig{}
				cfg.Connectors[component.NewID("nop/2")] = &errConfig{}
				pipe := cfg.Service.Pipelines[component.NewID("traces")]
				pipe.Receivers = append(pipe.Receivers, component.NewIDWithName("nop", "2"))
				pipe.Exporters = append(pipe.Exporters, component.NewIDWithName("nop", "2"))
				return cfg
			},
			expected: errors.New(`connectors::nop/2: there's already an exporter named "nop/2"`),
		},
		{
			name: "invalid-connector-reference-as-receiver",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[component.NewID("traces")]
				pipe.Receivers = append(pipe.Receivers, component.NewIDWithName("nop", "conn2"))
				return cfg
			},
			expected: errors.New(`service::pipeline::traces: references receiver "nop/conn2" which is not configured`),
		},
		{
			name: "invalid-connector-reference-as-receiver",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[component.NewID("traces")]
				pipe.Exporters = append(pipe.Exporters, component.NewIDWithName("nop", "conn2"))
				return cfg
			},
			expected: errors.New(`service::pipeline::traces: references exporter "nop/conn2" which is not configured`),
		},
		{
			name: "missing-connector-as-receiver",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[component.NewID("traces")]
				pipe.Exporters = append(pipe.Exporters, component.NewIDWithName("nop", "conn"))
				return cfg
			},
			expected: errors.New(`connectors::nop/conn: must be used as both receiver and exporter but is not used as receiver`),
		},
		{
			name: "missing-connector-as-exporter",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[component.NewID("traces")]
				pipe.Receivers = append(pipe.Receivers, component.NewIDWithName("nop", "conn"))
				return cfg
			},
			expected: errors.New(`connectors::nop/conn: must be used as both receiver and exporter but is not used as exporter`),
		},
		{
			name: "invalid-service-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Service.Pipelines = nil
				return cfg
			},
			expected: errors.New(`service must have at least one pipeline`),
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
			component.NewID("nop"): &errConfig{},
		},
		Exporters: map[component.ID]component.Config{
			component.NewID("nop"): &errConfig{},
		},
		Processors: map[component.ID]component.Config{
			component.NewID("nop"): &errConfig{},
		},
		Connectors: map[component.ID]component.Config{
			component.NewIDWithName("nop", "conn"): &errConfig{},
		},
		Extensions: map[component.ID]component.Config{
			component.NewID("nop"): &errConfig{},
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
			Extensions: []component.ID{component.NewID("nop")},
			Pipelines: map[component.ID]*service.PipelineConfig{
				component.NewID("traces"): {
					Receivers:  []component.ID{component.NewID("nop")},
					Processors: []component.ID{component.NewID("nop")},
					Exporters:  []component.ID{component.NewID("nop")},
				},
			},
		},
	}
}
