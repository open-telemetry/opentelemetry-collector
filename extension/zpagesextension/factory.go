// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpagesextension // import "go.opentelemetry.io/collector/extension/zpagesextension"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/zpagesextension/internal/metadata"
)

const (
	defaultEndpoint                 = "localhost:55679"
	defaultSupportBundleArchiveName = "otelcol-support-bundle"
)

// NewFactory creates a factory for Z-Pages extension.
func NewFactory() extension.Factory {
	return extension.NewFactory(metadata.Type, createDefaultConfig, create, metadata.ExtensionStability)
}

func createDefaultConfig() component.Config {
	serverConfig := confighttp.NewDefaultServerConfig()
	serverConfig.NetAddr.Endpoint = defaultEndpoint
	return &Config{
		ServerConfig: serverConfig,
		SupportBundle: configoptional.Default(SupportBundleConfig{
			ArchiveName: defaultSupportBundleArchiveName,
			Gatherers: GatherersConfig{
				PProf: GathererConfig{Enabled: true},
			},
		}),
	}
}

// create creates the extension based on this config.
func create(_ context.Context, set extension.Settings, cfg component.Config) (extension.Extension, error) {
	return newServer(cfg.(*Config), set.TelemetrySettings, set.BuildInfo), nil
}
