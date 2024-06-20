// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcoltest // import "go.opentelemetry.io/collector/otelcol/otelcoltest"

import (
	"context"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/converter/expandconverter"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/httpprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/otelcol"
)

// LoadConfigWithSettings loads a config.Config from the provider settings, and does NOT validate the configuration.
func LoadConfigWithSettings(factories otelcol.Factories, set otelcol.ConfigProviderSettings) (*otelcol.Config, error) {
	// Read yaml config from file
	provider, err := otelcol.NewConfigProvider(set)
	if err != nil {
		return nil, err
	}
	return provider.Get(context.Background(), factories)
}

// LoadConfig loads a config.Config  from file, and does NOT validate the configuration.
func LoadConfig(fileName string, factories otelcol.Factories) (*otelcol.Config, error) {
	return LoadConfigWithSettings(factories, otelcol.ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs: []string{fileName},
			ProviderFactories: []confmap.ProviderFactory{
				fileprovider.NewFactory(),
				envprovider.NewFactory(),
				yamlprovider.NewFactory(),
				httpprovider.NewFactory(),
			},
			ConverterFactories: []confmap.ConverterFactory{expandconverter.NewFactory()},
		},
	})
}

// LoadConfigAndValidateWithSettings loads a config from the provider settings, and validates the configuration.
func LoadConfigAndValidateWithSettings(factories otelcol.Factories, set otelcol.ConfigProviderSettings) (*otelcol.Config, error) {
	cfg, err := LoadConfigWithSettings(factories, set)
	if err != nil {
		return nil, err
	}
	return cfg, cfg.Validate()
}

// LoadConfigAndValidate loads a config from the file, and validates the configuration.
func LoadConfigAndValidate(fileName string, factories otelcol.Factories) (*otelcol.Config, error) {
	cfg, err := LoadConfigWithSettings(factories, otelcol.ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs: []string{fileName},
			ProviderFactories: []confmap.ProviderFactory{
				fileprovider.NewFactory(),
				envprovider.NewFactory(),
				yamlprovider.NewFactory(),
				httpprovider.NewFactory(),
			},
			ConverterFactories: []confmap.ConverterFactory{expandconverter.NewFactory()},
		},
	})
	if err != nil {
		return nil, err
	}
	return cfg, cfg.Validate()
}
