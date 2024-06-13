// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/otelcol/internal/configunmarshaler"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/telemetry"
	"go.uber.org/zap"
)

type configSettings struct {
	Receivers  *configunmarshaler.Configs[receiver.Factory]  `mapstructure:"receivers"`
	Processors *configunmarshaler.Configs[processor.Factory] `mapstructure:"processors"`
	Exporters  *configunmarshaler.Configs[exporter.Factory]  `mapstructure:"exporters"`
	Connectors *configunmarshaler.Configs[connector.Factory] `mapstructure:"connectors"`
	Extensions *configunmarshaler.Configs[extension.Factory] `mapstructure:"extensions"`
	Service    service.Config                                `mapstructure:"service"`
}

// unmarshal the configSettings from a confmap.Conf.
// After the config is unmarshalled, `Validate()` must be called to validate.
func unmarshal(logger *zap.Logger, v *confmap.Conf, factories Factories) (*configSettings, error) {

	telFactory := telemetry.NewFactory()
	defaultTelConfig := *telFactory.CreateDefaultConfig().(*telemetry.Config)

	// Unmarshal top level sections and validate.
	cfg := &configSettings{
		Receivers:  configunmarshaler.NewConfigs(logger, factories.Receivers),
		Processors: configunmarshaler.NewConfigs(logger, factories.Processors),
		Exporters:  configunmarshaler.NewConfigs(logger, factories.Exporters),
		Connectors: configunmarshaler.NewConfigs(logger, factories.Connectors),
		Extensions: configunmarshaler.NewConfigs(logger, factories.Extensions),
		// TODO: Add a component.ServiceFactory to allow this to be defined by the Service.
		Service: service.Config{
			Telemetry: defaultTelConfig,
		},
	}

	return cfg, v.Unmarshal(&cfg)
}
