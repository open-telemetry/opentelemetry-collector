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

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/service/telemetry"
)

var configNop = &Config{
	Receivers:  map[config.ComponentID]config.Receiver{config.NewComponentID("nop"): componenttest.NewNopReceiverFactory().CreateDefaultConfig()},
	Processors: map[config.ComponentID]config.Processor{config.NewComponentID("nop"): componenttest.NewNopProcessorFactory().CreateDefaultConfig()},
	Exporters:  map[config.ComponentID]config.Exporter{config.NewComponentID("nop"): componenttest.NewNopExporterFactory().CreateDefaultConfig()},
	Extensions: map[config.ComponentID]config.Extension{config.NewComponentID("nop"): componenttest.NewNopExtensionFactory().CreateDefaultConfig()},
	Service: ConfigService{
		Extensions: []config.ComponentID{config.NewComponentID("nop")},
		Pipelines: map[config.ComponentID]*ConfigServicePipeline{
			config.NewComponentID("traces"): {
				Receivers:  []config.ComponentID{config.NewComponentID("nop")},
				Processors: []config.ComponentID{config.NewComponentID("nop")},
				Exporters:  []config.ComponentID{config.NewComponentID("nop")},
			},
			config.NewComponentID("metrics"): {
				Receivers:  []config.ComponentID{config.NewComponentID("nop")},
				Processors: []config.ComponentID{config.NewComponentID("nop")},
				Exporters:  []config.ComponentID{config.NewComponentID("nop")},
			},
			config.NewComponentID("logs"): {
				Receivers:  []config.ComponentID{config.NewComponentID("nop")},
				Processors: []config.ComponentID{config.NewComponentID("nop")},
				Exporters:  []config.ComponentID{config.NewComponentID("nop")},
			},
		},
		Telemetry: telemetry.Config{
			Logs: telemetry.LogsConfig{
				Level:             zapcore.InfoLevel,
				Development:       false,
				Encoding:          "console",
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
