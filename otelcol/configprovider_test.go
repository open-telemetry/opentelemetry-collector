// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "go.yaml.in/yaml/v3"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/pipelines"
)

func TestConfigProvider(t *testing.T) {
	nopComponentID := component.MustNewID("nop")
	nopConComponentID := component.MustNewIDWithName("nop", "con")

	tests := map[string]struct {
		filename       string
		factories      func() (Factories, error)
		expectedConfig *Config
	}{
		"nop": {
			filename:  "otelcol-nop.yaml",
			factories: nopFactories,
			expectedConfig: &Config{
				Connectors: map[component.ID]component.Config{
					nopConComponentID: connectortest.NewNopFactory().CreateDefaultConfig(),
				},
				Exporters: map[component.ID]component.Config{
					nopComponentID: exportertest.NewNopFactory().CreateDefaultConfig(),
				},
				Extensions: map[component.ID]component.Config{
					nopComponentID: extensiontest.NewNopFactory().CreateDefaultConfig(),
				},
				Processors: map[component.ID]component.Config{
					nopComponentID: processortest.NewNopFactory().CreateDefaultConfig(),
				},
				Receivers: map[component.ID]component.Config{
					nopComponentID: receivertest.NewNopFactory().CreateDefaultConfig(),
				},
				Service: service.Config{
					Telemetry:  fakeTelemetryConfig{},
					Extensions: []component.ID{nopComponentID},
					Pipelines: map[pipeline.ID]*pipelines.PipelineConfig{
						pipeline.NewID(pipeline.SignalTraces): {
							Receivers:  []component.ID{nopComponentID},
							Processors: []component.ID{nopComponentID},
							Exporters:  []component.ID{nopComponentID, nopConComponentID},
						},
						pipeline.NewID(pipeline.SignalMetrics): {
							Receivers:  []component.ID{nopComponentID},
							Processors: []component.ID{nopComponentID},
							Exporters:  []component.ID{nopComponentID},
						},
						pipeline.NewID(pipeline.SignalLogs): {
							Receivers:  []component.ID{nopComponentID, nopConComponentID},
							Processors: []component.ID{nopComponentID},
							Exporters:  []component.ID{nopComponentID},
						},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			yamlBytes, err := os.ReadFile(filepath.Join("testdata", test.filename))
			require.NoError(t, err)

			yamlProvider := newFakeProvider("yaml", func(context.Context, string, confmap.WatcherFunc) (*confmap.Retrieved, error) {
				var rawConf any
				if yamlErr := yaml.Unmarshal(yamlBytes, &rawConf); yamlErr != nil {
					return nil, yamlErr
				}
				return confmap.NewRetrieved(rawConf)
			})

			set := ConfigProviderSettings{
				ResolverSettings: confmap.ResolverSettings{
					URIs:              []string{"yaml:" + string(yamlBytes)},
					ProviderFactories: []confmap.ProviderFactory{yamlProvider},
				},
			}

			cp, err := NewConfigProvider(set)
			require.NoError(t, err)

			factories, err := test.factories()
			require.NoError(t, err)

			cfg, err := cp.Get(context.Background(), factories)
			require.NoError(t, err)

			assert.Equal(t, test.expectedConfig, cfg)
		})
	}
}
