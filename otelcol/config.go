// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/pipelines"
)

var (
	errMissingExporters       = errors.New("no exporter configuration specified in config")
	errMissingReceivers       = errors.New("no receiver configuration specified in config")
	errEmptyConfigurationFile = errors.New("empty configuration file")
)

// Config defines the configuration for the various elements of collector or agent.
type Config struct {
	// Receivers is a map of ComponentID to Receivers.
	Receivers map[component.ID]component.Config `mapstructure:"receivers"`

	// Exporters is a map of ComponentID to Exporters.
	Exporters map[component.ID]component.Config `mapstructure:"exporters"`

	// Processors is a map of ComponentID to Processors.
	Processors map[component.ID]component.Config `mapstructure:"processors"`

	// Connectors is a map of ComponentID to connectors.
	Connectors map[component.ID]component.Config `mapstructure:"connectors"`

	// Extensions is a map of ComponentID to extensions.
	Extensions map[component.ID]component.Config `mapstructure:"extensions"`

	Service service.Config `mapstructure:"service"`

	// prevent unkeyed literal initialization
	_ struct{}
}

// Validate returns an error if the config is invalid.
//
// This function performs basic validation of configuration. There may be more subtle
// invalid cases that we currently don't check for but which we may want to add in
// the future (e.g. disallowing receiving and exporting on the same endpoint).
func (cfg *Config) Validate() error {
	// There must be at least one property set in the configuration	file.
	if len(cfg.Receivers) == 0 && len(cfg.Exporters) == 0 && len(cfg.Processors) == 0 && len(cfg.Connectors) == 0 && len(cfg.Extensions) == 0 {
		return errEmptyConfigurationFile
	}

	// Currently, there is no default receiver enabled.
	// The configuration must specify at least one receiver to be valid.
	if !pipelines.AllowNoPipelines.IsEnabled() && len(cfg.Receivers) == 0 {
		return errMissingReceivers
	}

	// Currently, there is no default exporter enabled.
	// The configuration must specify at least one exporter to be valid.
	if !pipelines.AllowNoPipelines.IsEnabled() && len(cfg.Exporters) == 0 {
		return errMissingExporters
	}

	// Validate the connector configuration.
	for connID := range cfg.Connectors {
		if _, ok := cfg.Exporters[connID]; ok {
			return fmt.Errorf("connectors::%s: ambiguous ID: Found both %q exporter and %q connector. "+
				"Change one of the components' IDs to eliminate ambiguity (e.g. rename %q connector to %q)",
				connID, connID, connID, connID, connID.String()+"/connector")
		}
		if _, ok := cfg.Receivers[connID]; ok {
			return fmt.Errorf("connectors::%s: ambiguous ID: Found both %q receiver and %q connector. "+
				"Change one of the components' IDs to eliminate ambiguity (e.g. rename %q connector to %q)",
				connID, connID, connID, connID, connID.String()+"/connector")
		}
	}

	// Check that all enabled extensions in the service are configured.
	for _, ref := range cfg.Service.Extensions {
		// Check that the name referenced in the Service extensions exists in the top-level extensions.
		if cfg.Extensions[ref] == nil {
			return fmt.Errorf("service::extensions: references extension %q which is not configured", ref)
		}
	}

	// Check that all pipelines reference only configured components.
	for pipelineID, pipeline := range cfg.Service.Pipelines {
		// Validate pipeline receiver name references.
		for _, ref := range pipeline.Receivers {
			// Check that the name referenced in the pipeline's receivers exists in the top-level receivers.
			if _, ok := cfg.Receivers[ref]; ok {
				continue
			}

			if _, ok := cfg.Connectors[ref]; ok {
				continue
			}
			return fmt.Errorf("service::pipelines::%s: references receiver %q which is not configured", pipelineID.String(), ref)
		}

		// Validate pipeline processor name references.
		for _, ref := range pipeline.Processors {
			// Check that the name referenced in the pipeline's processors exists in the top-level processors.
			if cfg.Processors[ref] == nil {
				return fmt.Errorf("service::pipelines::%s: references processor %q which is not configured", pipelineID.String(), ref)
			}
		}

		// Validate pipeline exporter name references.
		for _, ref := range pipeline.Exporters {
			// Check that the name referenced in the pipeline's Exporters exists in the top-level Exporters.
			if _, ok := cfg.Exporters[ref]; ok {
				continue
			}
			if _, ok := cfg.Connectors[ref]; ok {
				continue
			}
			return fmt.Errorf("service::pipelines::%s: references exporter %q which is not configured", pipelineID.String(), ref)
		}
	}
	return nil
}
