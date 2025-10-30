// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcoltest // import "go.opentelemetry.io/collector/otelcol/otelcoltest"

import (
	"context"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/httpprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/otelcol"
)

// LoadConfig loads a config.Config from file, and does NOT validate the configuration.
//
// If factories.Telemetry is nil, a no-op telemetry factory will be used. This
// factory does not support any telemetry configuration.
func LoadConfig(fileName string, factories otelcol.Factories) (*otelcol.Config, error) {
	if factories.Telemetry == nil {
		factories.Telemetry = nopTelemetryFactory()
	}
	provider, err := otelcol.NewConfigProvider(otelcol.ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs: []string{fileName},
			ProviderFactories: []confmap.ProviderFactory{
				fileprovider.NewFactory(),
				envprovider.NewFactory(),
				yamlprovider.NewFactory(),
				httpprovider.NewFactory(),
			},
			DefaultScheme: "env",
		},
	})
	if err != nil {
		return nil, err
	}
	return provider.Get(context.Background(), factories)
}

// LoadConfigAndValidate loads a config from the file, and validates the configuration.
func LoadConfigAndValidate(fileName string, factories otelcol.Factories) (*otelcol.Config, error) {
	cfg, err := LoadConfig(fileName, factories)
	if err != nil {
		return nil, err
	}
	return cfg, xconfmap.Validate(cfg)
}
