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
	"go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"
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
func unmarshal(v *confmap.Conf, factories Factories) (*configSettings, error) {
	// TODO remove these params once SDK and resource creation are encapsulated
	// within the otelconftelemetry factory. They are not used when creating
	// the default config.
	// See https://github.com/open-telemetry/opentelemetry-collector/issues/4970
	telFactory := otelconftelemetry.NewFactory(nil, nil)
	defaultTelConfig := *telFactory.CreateDefaultConfig().(*otelconftelemetry.Config)

	// Unmarshal top level sections and validate.
	cfg := &configSettings{
		Receivers:  configunmarshaler.NewConfigs(factories.Receivers),
		Processors: configunmarshaler.NewConfigs(factories.Processors),
		Exporters:  configunmarshaler.NewConfigs(factories.Exporters),
		Connectors: configunmarshaler.NewConfigs(factories.Connectors),
		Extensions: configunmarshaler.NewConfigs(factories.Extensions),
		// TODO: Add a component.ServiceFactory to allow this to be defined by the Service.
		Service: service.Config{
			Telemetry: defaultTelConfig,
		},
	}
	err := v.Unmarshal(&cfg)
	return cfg, err
}
