// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"fmt"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/otelcol/internal/configunmarshaler"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/telemetry"
)

type configSettings struct {
	Receivers  *configunmarshaler.Configs[receiver.Factory]  `mapstructure:"receivers"`
	Processors *configunmarshaler.Configs[processor.Factory] `mapstructure:"processors"`
	Exporters  *configunmarshaler.Configs[exporter.Factory]  `mapstructure:"exporters"`
	Connectors *configunmarshaler.Configs[connector.Factory] `mapstructure:"connectors"`
	Extensions *configunmarshaler.Configs[extension.Factory] `mapstructure:"extensions"`
	Service    service.Config                                `mapstructure:"service"`
}

func (c *configSettings) Validate() error {
	receivers := c.Receivers.Configs()
	processors := c.Processors.Configs()
	exporters := c.Exporters.Configs()
	connectors := c.Connectors.Configs()
	extensions := c.Extensions.Configs()

	// There must be at least one property set in the configuration	file.
	if len(receivers) == 0 && len(exporters) == 0 && len(processors) == 0 && len(connectors) == 0 && len(extensions) == 0 {
		return errEmptyConfigurationFile
	}

	// Currently, there is no default receiver enabled.
	// The configuration must specify at least one receiver to be valid.
	if len(receivers) == 0 {
		return errMissingReceivers
	}

	// Currently, there is no default exporter enabled.
	// The configuration must specify at least one exporter to be valid.
	if len(exporters) == 0 {
		return errMissingExporters
	}

	// Validate the connector configuration.
	for connID, _ := range connectors {
		if _, ok := exporters[connID]; ok {
			return fmt.Errorf("connectors::%s: ambiguous ID: Found both %q exporter and %q connector. "+
				"Change one of the components' IDs to eliminate ambiguity (e.g. rename %q connector to %q)",
				connID, connID, connID, connID, connID.String()+"/connector")
		}
		if _, ok := receivers[connID]; ok {
			return fmt.Errorf("connectors::%s: ambiguous ID: Found both %q receiver and %q connector. "+
				"Change one of the components' IDs to eliminate ambiguity (e.g. rename %q connector to %q)",
				connID, connID, connID, connID, connID.String()+"/connector")
		}
	}

	// Check that all enabled extensions in the service are configured.
	for _, ref := range c.Service.Extensions {
		// Check that the name referenced in the Service extensions exists in the top-level extensions.
		if extensions[ref] == nil {
			return fmt.Errorf("service::extensions: references extension %q which is not configured", ref)
		}
	}

	// Check that all pipelines reference only configured components.
	for pipelineID, pipeline := range c.Service.Pipelines {
		// Validate pipeline receiver name references.
		for _, ref := range pipeline.Receivers {
			// Check that the name referenced in the pipeline's receivers exists in the top-level receivers.
			if _, ok := receivers[ref]; ok {
				continue
			}

			if _, ok := connectors[ref]; ok {
				continue
			}
			return fmt.Errorf("service::pipelines::%s: references receiver %q which is not configured", pipelineID.String(), ref)
		}

		// Validate pipeline processor name references.
		for _, ref := range pipeline.Processors {
			// Check that the name referenced in the pipeline's processors exists in the top-level processors.
			if processors[ref] == nil {
				return fmt.Errorf("service::pipelines::%s: references processor %q which is not configured", pipelineID.String(), ref)
			}
		}

		// Validate pipeline exporter name references.
		for _, ref := range pipeline.Exporters {
			// Check that the name referenced in the pipeline's Exporters exists in the top-level Exporters.
			if _, ok := exporters[ref]; ok {
				continue
			}
			if _, ok := connectors[ref]; ok {
				continue
			}
			return fmt.Errorf("service::pipelines::%s: references exporter %q which is not configured", pipelineID.String(), ref)
		}
	}
	return nil
}

// unmarshal the configSettings from a confmap.Conf.
// After the config is unmarshalled, `Validate()` must be called to validate.
func unmarshal(v *confmap.Conf, factories Factories) (*configSettings, error) {
	telFactory := telemetry.NewFactory()
	defaultTelConfig := *telFactory.CreateDefaultConfig().(*telemetry.Config)

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

	return cfg, v.Unmarshal(&cfg, confmap.WithInvokeValidate())
}
