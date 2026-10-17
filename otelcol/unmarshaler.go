// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"errors"

	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/otelcol/internal/configunmarshaler"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service"
)

var errNilTelemetryFactory = errors.New("otelcol.Factories.Telemetry must not be nil. For example, you can use otelconftelemetry.NewFactory to build a telemetry factory")

type configSettings struct {
	Receivers  *configunmarshaler.Configs[receiver.Factory]  `mapstructure:"receivers"`
	Processors *configunmarshaler.Configs[processor.Factory] `mapstructure:"processors"`
	Exporters  *configunmarshaler.Configs[exporter.Factory]  `mapstructure:"exporters"`
	Connectors *configunmarshaler.Configs[connector.Factory] `mapstructure:"connectors"`
	Extensions *configunmarshaler.Configs[extension.Factory] `mapstructure:"extensions"`
	Service    service.Config                                `mapstructure:"service"`

	// ComponentLogLevelOverrides collects per-component log level overrides
	// from all component configs. Populated after Unmarshal.
	ComponentLogLevelOverrides map[component.ID]*zapcore.Level `mapstructure:"-"`
}

// unmarshal the configSettings from a confmap.Conf.
// After the config is unmarshalled, `Validate()` must be called to validate.
func unmarshal(v *confmap.Conf, factories Factories) (*configSettings, error) {
	if factories.Telemetry == nil {
		return nil, errNilTelemetryFactory
	}

	// Unmarshal top level sections and validate.
	cfg := &configSettings{
		Receivers:  configunmarshaler.NewConfigs(factories.Receivers),
		Processors: configunmarshaler.NewConfigs(factories.Processors),
		Exporters:  configunmarshaler.NewConfigs(factories.Exporters),
		Connectors: configunmarshaler.NewConfigs(factories.Connectors),
		Extensions: configunmarshaler.NewConfigs(factories.Extensions),
		// TODO: Add a component.ServiceFactory to allow this to be defined by the Service.
		Service: service.Config{
			Telemetry: factories.Telemetry.CreateDefaultConfig(),
		},
	}
	err := v.Unmarshal(&cfg)
	if err != nil {
		return cfg, err
	}

	// Collect per-component log level overrides from all component kinds.
	cfg.ComponentLogLevelOverrides = collectLogLevelOverrides(
		cfg.Receivers.TelemetryOverrides(),
		cfg.Processors.TelemetryOverrides(),
		cfg.Exporters.TelemetryOverrides(),
		cfg.Connectors.TelemetryOverrides(),
		cfg.Extensions.TelemetryOverrides(),
	)

	return cfg, nil
}

// collectLogLevelOverrides merges log level overrides from multiple component kinds
// into a single map keyed by component.ID.
func collectLogLevelOverrides(overrides ...map[component.ID]configunmarshaler.TelemetryOverride) map[component.ID]*zapcore.Level {
	var result map[component.ID]*zapcore.Level
	for _, ovMap := range overrides {
		for id, ov := range ovMap {
			if ov.LogLevel != nil {
				if result == nil {
					result = make(map[component.ID]*zapcore.Level)
				}
				result[id] = ov.LogLevel
			}
		}
	}
	return result
}
