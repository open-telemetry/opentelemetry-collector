// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpagesextension // import "go.opentelemetry.io/collector/extension/zpagesextension"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/zpagesextension/internal/metadata"
)

const (
	defaultEndpoint = "localhost:55679"
)

// NewFactory creates a factory for Z-Pages extension.
func NewFactory() extension.Factory {
	return extension.NewFactory(metadata.Type, createDefaultConfig, create, metadata.ExtensionStability)
}

func createDefaultConfig() component.Config {
	return &Config{
		ServerConfig: confighttp.ServerConfig{
			Endpoint: defaultEndpoint,
		},
	}
}

// create creates the extension based on this config.
func create(_ context.Context, set extension.Settings, cfg component.Config) (extension.Extension, error) {
	return newServer(cfg.(*Config), set.TelemetrySettings), nil
}
