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
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/service/telemetry"
)

var configNop = &Config{
	Receivers:  map[component.ID]component.ReceiverConfig{component.NewID("nop"): componenttest.NewNopReceiverFactory().CreateDefaultConfig()},
	Processors: map[component.ID]component.ProcessorConfig{component.NewID("nop"): componenttest.NewNopProcessorFactory().CreateDefaultConfig()},
	Exporters:  map[component.ID]component.ExporterConfig{component.NewID("nop"): componenttest.NewNopExporterFactory().CreateDefaultConfig()},
	Extensions: map[component.ID]component.ExtensionConfig{component.NewID("nop"): componenttest.NewNopExtensionFactory().CreateDefaultConfig()},
	Service: ConfigService{
		Extensions: []component.ID{component.NewID("nop")},
		Pipelines: map[component.ID]*ConfigServicePipeline{
			component.NewID("traces"): {
				Receivers:  []component.ID{component.NewID("nop")},
				Processors: []component.ID{component.NewID("nop")},
				Exporters:  []component.ID{component.NewID("nop")},
			},
			component.NewID("metrics"): {
				Receivers:  []component.ID{component.NewID("nop")},
				Processors: []component.ID{component.NewID("nop")},
				Exporters:  []component.ID{component.NewID("nop")},
			},
			component.NewID("logs"): {
				Receivers:  []component.ID{component.NewID("nop")},
				Processors: []component.ID{component.NewID("nop")},
				Exporters:  []component.ID{component.NewID("nop")},
			},
		},
		Telemetry: telemetry.Config{
			Logs: telemetry.LogsConfig{
				Level:       zapcore.InfoLevel,
				Development: false,
				Encoding:    "console",
				Sampling: &telemetry.LogsSamplingConfig{
					Initial:    100,
					Thereafter: 100,
				},
				OutputPaths:       []string{"stderr"},
				ErrorOutputPaths:  []string{"stderr"},
				DisableCaller:     false,
				DisableStacktrace: false,
				InitialFields:     map[string]interface{}(nil),
			},
			Metrics: telemetry.MetricsConfig{
				Level:   configtelemetry.LevelBasic,
				Address: "localhost:8888",
			},
		},
	},
}

func TestConfigProviderYaml(t *testing.T) {
	yamlBytes, err := os.ReadFile(filepath.Join("testdata", "otelcol-nop.yaml"))
	require.NoError(t, err)

	uriLocation := "yaml:" + string(yamlBytes)
	provider := yamlprovider.New()
	set := ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs:      []string{uriLocation},
			Providers: map[string]confmap.Provider{provider.Scheme(): provider},
		},
	}

	cp, err := NewConfigProvider(set)
	require.NoError(t, err)

	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	cfg, err := cp.Get(context.Background(), factories)
	require.NoError(t, err)
	assert.EqualValues(t, configNop, cfg)
}

func TestConfigProviderFile(t *testing.T) {
	uriLocation := "file:" + filepath.Join("testdata", "otelcol-nop.yaml")
	provider := fileprovider.New()
	set := ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs:      []string{uriLocation},
			Providers: map[string]confmap.Provider{provider.Scheme(): provider},
		},
	}

	cp, err := NewConfigProvider(set)
	require.NoError(t, err)

	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	cfg, err := cp.Get(context.Background(), factories)
	require.NoError(t, err)
	assert.EqualValues(t, configNop, cfg)
}
