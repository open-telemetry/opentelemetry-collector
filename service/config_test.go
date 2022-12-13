// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestConfigDryValidate(t *testing.T) {
	var testCases = []struct {
		name     string // test case name (also file name containing config yaml)
		cfgFn    func() *Config
		expected string
	}{
		{
			name:     "valid",
			cfgFn:    generateConfig,
			expected: "",
		},
		{
			name: "missing-exporters, invalid-receiver-reference, invalid-extension-reference",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Exporters = nil
				pipe := cfg.Service.Pipelines[component.NewID("traces")]
				pipe.Receivers = append(pipe.Receivers, component.NewIDWithName("nop", "2"))
				cfg.Service.Extensions = append(cfg.Service.Extensions, component.NewIDWithName("nop", "2"))
				return cfg
			},
			expected: fmt.Sprintf(
				`**..%v
**..service::extensions: references extension "nop/2" which is not configured
**..service::pipeline::traces: references receiver "nop/2" which is not configured
**..service::pipeline::traces: references exporter "nop" which is not configured`, errMissingExporters),
		},
		{
			name: "missing-receivers, invalid-exporter-config, missing-pipeline-receivers",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Receivers = nil
				cfg.Exporters[component.NewID("nop")] = &nopExpConfig{
					validateErr: errInvalidExpConfig,
				}
				pipe := cfg.Service.Pipelines[component.NewID("traces")]
				pipe.Receivers = nil
				return cfg
			},
			expected: fmt.Sprintf(`**..%v
**..exporters::nop: %v
**..service::pipeline::traces: %v`, errMissingReceivers, errInvalidExpConfig, errMissingServicePipelineReceivers),
		},
		{
			name: "invalid-exporter-reference, invalid-processor-config, invalid-receiver-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				cfg.Processors[component.NewID("nop")] = &nopProcConfig{
					validateErr: errInvalidProcConfig,
				}
				cfg.Receivers[component.NewID("nop")] = &nopRecvConfig{
					validateErr: errInvalidRecvConfig,
				}
				pipe := cfg.Service.Pipelines[component.NewID("traces")]
				pipe.Exporters = append(pipe.Exporters, component.NewIDWithName("nop", "2"))
				return cfg
			},
			expected: fmt.Sprintf(`**..receivers::nop: %v
**..processors::nop: %v
**..service::pipeline::traces: references exporter "nop/2" which is not configured`, errInvalidRecvConfig, errInvalidProcConfig),
		},
		{
			name: "missing-pipeline-exporters, invalid-extension-config",
			cfgFn: func() *Config {
				cfg := generateConfig()
				pipe := cfg.Service.Pipelines[component.NewID("traces")]
				pipe.Exporters = nil
				cfg.Extensions[component.NewID("nop")] = &nopExtConfig{
					validateErr: errInvalidExtConfig,
				}
				return cfg
			},
			expected: fmt.Sprintf(`**..extensions::nop: %v
**..service::pipeline::traces: %v`, errInvalidExtConfig, errMissingServicePipelineExporters),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			cfg := test.cfgFn()
			oldStdOut := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w // Write to os.StdOut

			cfg.DryRunValidate()

			bufChan := make(chan string)

			go func() {
				var buf bytes.Buffer
				_, err := io.Copy(&buf, r)
				require.NoError(t, err)
				bufChan <- buf.String()
			}()

			err := w.Close()
			require.NoError(t, err)
			defer func() { os.Stdout = oldStdOut }() // Restore os.Stdout to old value after test
			output := <-bufChan
			assert.Equal(t, test.expected, strings.Trim(output, "\n"))
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
