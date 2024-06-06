// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcoltest // import "go.opentelemetry.io/collector/otelcol/otelcoltest"

import (
	"context"

	"go.opentelemetry.io/collector/otelcol"
)

// LoadConfig loads a config.Config  from file, and does NOT validate the configuration.
func LoadConfig(fileName string, factories otelcol.Factories, set otelcol.ConfigProviderSettings) (*otelcol.Config, error) {
	// Read yaml config from file
	provider, err := otelcol.NewConfigProvider(set)
	if err != nil {
		return nil, err
	}
	return provider.Get(context.Background(), factories)
}

// LoadConfigAndValidate loads a config from the file, and validates the configuration.
func LoadConfigAndValidate(fileName string, factories otelcol.Factories, set otelcol.ConfigProviderSettings) (*otelcol.Config, error) {
	cfg, err := LoadConfig(fileName, factories, set)
	if err != nil {
		return nil, err
	}
	return cfg, cfg.Validate()
}
