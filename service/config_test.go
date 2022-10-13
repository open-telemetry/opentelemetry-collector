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

package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/service/telemetry"
)

var (
	errMissingExporters        = errors.New("no enabled exporters specified in config")
	errMissingReceivers        = errors.New("no enabled receivers specified in config")
	errMissingServicePipelines = errors.New("service must have at least one pipeline")

	errInvalidRecvConfig = errors.New("invalid receiver config")
	errInvalidExpConfig  = errors.New("invalid exporter config")
	errInvalidProcConfig = errors.New("invalid processor config")
	errInvalidExtConfig  = errors.New("invalid extension config")
)

type nopRecvConfig struct {
	config.ReceiverSettings
	validateErr error
}

func (nc *nopRecvConfig) Validate() error {
	return nc.validateErr
}

type nopExpConfig struct {
	config.ExporterSettings
	validateErr error
}

func (nc *nopExpConfig) Validate() error {
	return nc.validateErr
}

type nopProcConfig struct {
	config.ProcessorSettings
	validateErr error
}

func (nc *nopProcConfig) Validate() error {
	return nc.validateErr
}

type nopExtConfig struct {
	config.ExtensionSettings
	validateErr error
}

func (nc *nopExtConfig) Validate() error {
	return nc.validateErr
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
				cfg.Service.Telemetry.Logs.Encoding = "test_encoding"
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
				cfg.Service.Extensions = append(cfg.Service.Extensions, config.NewComponentIDWithName("nop", "2"))
				return cfg
			},
			expected: errors.New(`service references extension "nop/2" which does not exist`),
		},
		{
			name: "invalid-receiver-reference",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[config.NewComponentID("traces")]
				pipe.Receivers = append(pipe.Receivers, config.NewComponentIDWithName("nop", "2"))
				return cfg
			},
			expected: errors.New(`pipeline "traces" references receiver "nop/2" which does not exist`),
		},
		{
			name: "invalid-processor-reference",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[config.NewComponentID("traces")]
				pipe.Processors = append(pipe.Processors, config.NewComponentIDWithName("nop", "2"))
				return cfg
			},
			expected: errors.New(`pipeline "traces" references processor "nop/2" which does not exist`),
		},
		{
			name: "invalid-exporter-reference",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[config.NewComponentID("traces")]
				pipe.Exporters = append(pipe.Exporters, config.NewComponentIDWithName("nop", "2"))
				return cfg
			},
			expected: errors.New(`pipeline "traces" references exporter "nop/2" which does not exist`),
		},
		{
			name: "missing-pipeline-receivers",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[config.NewComponentID("traces")]
				pipe.Receivers = nil
				return cfg
			},
			expected: errors.New(`pipeline "traces" must have at least one receiver`),
		},
		{
			name: "missing-pipeline-exporters",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[config.NewComponentID("traces")]
				pipe.Exporters = nil
				return cfg
			},
			expected: errors.New(`pipeline "traces" must have at least one exporter`),
		},
		{
			name: "missing-pipelines",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Service.Pipelines = nil
				return cfg
			},
			expected: errMissingServicePipelines,
		},
		{
			name: "invalid-receiver-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Receivers[config.NewComponentID("nop")] = &nopRecvConfig{
					ReceiverSettings: config.NewReceiverSettings(config.NewComponentID("nop")),
					validateErr:      errInvalidRecvConfig,
				}
				return cfg
			},
			expected: fmt.Errorf(`receiver "nop" has invalid configuration: %w`, errInvalidRecvConfig),
		},
		{
			name: "invalid-exporter-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Exporters[config.NewComponentID("nop")] = &nopExpConfig{
					ExporterSettings: config.NewExporterSettings(config.NewComponentID("nop")),
					validateErr:      errInvalidExpConfig,
				}
				return cfg
			},
			expected: fmt.Errorf(`exporter "nop" has invalid configuration: %w`, errInvalidExpConfig),
		},
		{
			name: "invalid-processor-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Processors[config.NewComponentID("nop")] = &nopProcConfig{
					ProcessorSettings: config.NewProcessorSettings(config.NewComponentID("nop")),
					validateErr:       errInvalidProcConfig,
				}
				return cfg
			},
			expected: fmt.Errorf(`processor "nop" has invalid configuration: %w`, errInvalidProcConfig),
		},
		{
			name: "invalid-extension-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Extensions[config.NewComponentID("nop")] = &nopExtConfig{
					ExtensionSettings: config.NewExtensionSettings(config.NewComponentID("nop")),
					validateErr:       errInvalidExtConfig,
				}
				return cfg
			},
			expected: fmt.Errorf(`extension "nop" has invalid configuration: %w`, errInvalidExtConfig),
		},
		{
			name: "invalid-service-pipeline-type",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Service.Pipelines[config.NewComponentID("wrongtype")] = &ConfigServicePipeline{
					Receivers:  []config.ComponentID{config.NewComponentID("nop")},
					Processors: []config.ComponentID{config.NewComponentID("nop")},
					Exporters:  []config.ComponentID{config.NewComponentID("nop")},
				}
				return cfg
			},
			expected: errors.New(`unknown pipeline datatype "wrongtype" for wrongtype`),
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
		Receivers: map[config.ComponentID]config.Receiver{
			config.NewComponentID("nop"): &nopRecvConfig{
				ReceiverSettings: config.NewReceiverSettings(config.NewComponentID("nop")),
			},
		},
		Exporters: map[config.ComponentID]config.Exporter{
			config.NewComponentID("nop"): &nopExpConfig{
				ExporterSettings: config.NewExporterSettings(config.NewComponentID("nop")),
			},
		},
		Processors: map[config.ComponentID]config.Processor{
			config.NewComponentID("nop"): &nopProcConfig{
				ProcessorSettings: config.NewProcessorSettings(config.NewComponentID("nop")),
			},
		},
		Extensions: map[config.ComponentID]config.Extension{
			config.NewComponentID("nop"): &nopExtConfig{
				ExtensionSettings: config.NewExtensionSettings(config.NewComponentID("nop")),
			},
		},
		Service: ConfigService{
			Telemetry: telemetry.Config{
				Logs: telemetry.LogsConfig{
					Level:             zapcore.DebugLevel,
					Development:       true,
					Encoding:          "console",
					DisableCaller:     true,
					DisableStacktrace: true,
					OutputPaths:       []string{"stderr", "./output-logs"},
					ErrorOutputPaths:  []string{"stderr", "./error-output-logs"},
					InitialFields:     map[string]interface{}{"fieldKey": "filed-value"},
				},
				Metrics: telemetry.MetricsConfig{
					Level:   configtelemetry.LevelNormal,
					Address: ":8080",
				},
			},
			Extensions: []config.ComponentID{config.NewComponentID("nop")},
			Pipelines: map[config.ComponentID]*ConfigServicePipeline{
				config.NewComponentID("traces"): {
					Receivers:  []config.ComponentID{config.NewComponentID("nop")},
					Processors: []config.ComponentID{config.NewComponentID("nop")},
					Exporters:  []config.ComponentID{config.NewComponentID("nop")},
				},
			},
		},
	}
}
