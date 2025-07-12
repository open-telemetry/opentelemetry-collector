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

	"go.opentelemetry.io/collector/confmap"
)

func newConfig(yamlBytes []byte, factories Factories) (*Config, error) {
	stringMap := map[string]any{}
	err := yaml.Unmarshal(yamlBytes, stringMap)
	if err != nil {
		return nil, err
	}

	conf := confmap.NewFromStringMap(stringMap)

	cfg, err := unmarshal(conf, factories)
	if err != nil {
		return nil, err
	}

	return &Config{
		Receivers:  cfg.Receivers.Configs(),
		Processors: cfg.Processors.Configs(),
		Exporters:  cfg.Exporters.Configs(),
		Connectors: cfg.Connectors.Configs(),
		Extensions: cfg.Extensions.Configs(),
		Service:    cfg.Service,
	}, nil
}

func TestConfigProviderYaml(t *testing.T) {
	yamlBytes, err := os.ReadFile(filepath.Join("testdata", "otelcol-nop.yaml"))
	require.NoError(t, err)

	uriLocation := "yaml:" + string(yamlBytes)

	yamlProvider := newFakeProvider("yaml", func(_ context.Context, _ string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
		var rawConf any
		if err = yaml.Unmarshal(yamlBytes, &rawConf); err != nil {
			return nil, err
		}
		return confmap.NewRetrieved(rawConf)
	})

	set := ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs:              []string{uriLocation},
			ProviderFactories: []confmap.ProviderFactory{yamlProvider},
		},
	}

	cp, err := NewConfigProvider(set)
	require.NoError(t, err)

	factories, err := nopFactories()
	require.NoError(t, err)

	cfg, err := cp.Get(context.Background(), factories)
	require.NoError(t, err)

	configNop, err := newConfig(yamlBytes, factories)
	require.NoError(t, err)

	assert.Equal(t, configNop, cfg)
}

func TestConfigProviderFile(t *testing.T) {
	uriLocation := "file:" + filepath.Join("testdata", "otelcol-nop.yaml")
	fileProvider := newFakeProvider("file", func(_ context.Context, _ string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
		return confmap.NewRetrieved(newConfFromFile(t, uriLocation[5:]))
	})
	set := ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs:              []string{uriLocation},
			ProviderFactories: []confmap.ProviderFactory{fileProvider},
		},
	}

	cp, err := NewConfigProvider(set)
	require.NoError(t, err)

	factories, err := nopFactories()
	require.NoError(t, err)

	cfg, err := cp.Get(context.Background(), factories)
	require.NoError(t, err)

	yamlBytes, err := os.ReadFile(filepath.Join("testdata", "otelcol-nop.yaml"))
	require.NoError(t, err)

	configNop, err := newConfig(yamlBytes, factories)
	require.NoError(t, err)

	assert.Equal(t, configNop, cfg)
}
