// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/service"
)

var (
	errMissingExporters = errors.New("no exporter configuration specified in config")
	errMissingReceivers = errors.New("no receiver configuration specified in config")
)

// Config defines the configuration for the various elements of collector or agent.
type Config struct {
	// Receivers is a map of ComponentID to Receivers.
	Receivers map[component.ID]component.Config

	// Exporters is a map of ComponentID to Exporters.
	Exporters map[component.ID]component.Config

	// Processors is a map of ComponentID to Processors.
	Processors map[component.ID]component.Config

	// Connectors is a map of ComponentID to connectors.
	Connectors map[component.ID]component.Config

	// Extensions is a map of ComponentID to extensions.
	Extensions map[component.ID]component.Config

	Service service.Config
}

// Validate returns an error if the config is invalid.
//
// This function performs basic validation of configuration. There may be more subtle
// invalid cases that we currently don't check for but which we may want to add in
// the future (e.g. disallowing receiving and exporting on the same endpoint).
func (cfg *Config) Validate(factories Factories) error {
	// Currently, there is no default receiver enabled.
	// The configuration must specify at least one receiver to be valid.
	if len(cfg.Receivers) == 0 {
		return errMissingReceivers
	}

	// Validate the receiver configuration.
	for recvID, recvCfg := range cfg.Receivers {
		if err := component.ValidateConfig(recvCfg); err != nil {
			return fmt.Errorf("receivers::%s: %w", recvID, err)
		}
	}

	// Currently, there is no default exporter enabled.
	// The configuration must specify at least one exporter to be valid.
	if len(cfg.Exporters) == 0 {
		return errMissingExporters
	}

	// Validate the exporter configuration.
	for expID, expCfg := range cfg.Exporters {
		if err := component.ValidateConfig(expCfg); err != nil {
			return fmt.Errorf("exporters::%s: %w", expID, err)
		}
	}

	// Validate the processor configuration.
	for procID, procCfg := range cfg.Processors {
		if err := component.ValidateConfig(procCfg); err != nil {
			return fmt.Errorf("processors::%s: %w", procID, err)
		}
	}

	// Validate the connector configuration.
	for connID, connCfg := range cfg.Connectors {
		if err := component.ValidateConfig(connCfg); err != nil {
			return fmt.Errorf("connectors::%s: %w", connID, err)
		}

		if _, ok := cfg.Exporters[connID]; ok {
			return fmt.Errorf("connectors::%s: there's already an exporter named %q", connID, connID)
		}
		if _, ok := cfg.Receivers[connID]; ok {
			return fmt.Errorf("connectors::%s: there's already a receiver named %q", connID, connID)
		}
	}

	// Validate the extension configuration.
	for extID, extCfg := range cfg.Extensions {
		if err := component.ValidateConfig(extCfg); err != nil {
			return fmt.Errorf("extensions::%s: %w", extID, err)
		}
	}

	if err := cfg.Service.Validate(); err != nil {
		return err
	}

	// Check that all enabled extensions in the service are configured.
	for _, ref := range cfg.Service.Extensions {
		// Check that the name referenced in the Service extensions exists in the top-level extensions.
		if cfg.Extensions[ref] == nil {
			return fmt.Errorf("service::extensions: references extension %q which is not configured", ref)
		}
	}

	// Keep track of whether connectors are used as receivers and exporters using map[connectorID][]pipelineID
	connectorsAsExporters := make(map[component.ID][]component.ID, len(cfg.Connectors))
	connectorsAsReceivers := make(map[component.ID][]component.ID, len(cfg.Connectors))

	// Check that all pipelines reference only configured components.
	for pipelineID, pipeline := range cfg.Service.Pipelines {
		// Validate pipeline receiver name references.
		for _, ref := range pipeline.Receivers {
			// Check that the name referenced in the pipeline's receivers exists in the top-level receivers.
			if _, ok := cfg.Receivers[ref]; ok {
				continue
			}

			if _, ok := cfg.Connectors[ref]; ok {
				connectorsAsReceivers[ref] = append(connectorsAsReceivers[ref], pipelineID)
				continue
			}
			return fmt.Errorf("service::pipelines::%s: references receiver %q which is not configured", pipelineID, ref)
		}

		// Validate pipeline processor name references.
		for _, ref := range pipeline.Processors {
			// Check that the name referenced in the pipeline's processors exists in the top-level processors.
			if cfg.Processors[ref] == nil {
				return fmt.Errorf("service::pipelines::%s: references processor %q which is not configured", pipelineID, ref)
			}
		}

		// Validate pipeline exporter name references.
		for _, ref := range pipeline.Exporters {
			// Check that the name referenced in the pipeline's Exporters exists in the top-level Exporters.
			if _, ok := cfg.Exporters[ref]; ok {
				continue
			}
			if _, ok := cfg.Connectors[ref]; ok {
				connectorsAsExporters[ref] = append(connectorsAsExporters[ref], pipelineID)
				continue
			}
			return fmt.Errorf("service::pipelines::%s: references exporter %q which is not configured", pipelineID, ref)
		}
	}

	// Validate that connectors are used as both receiver and exporter
	for connID := range cfg.Connectors {
		// TODO must use connector factories to assess validity
		expPipelines, expOK := connectorsAsExporters[connID]
		recPipelines, recOK := connectorsAsReceivers[connID]
		if recOK && !expOK {
			return fmt.Errorf("connectors::%s: must be used as both receiver and exporter but is not used as exporter", connID)
		}
		if !recOK && expOK {
			return fmt.Errorf("connectors::%s: must be used as both receiver and exporter but is not used as receiver", connID)
		}
		factory := factories.Connectors[connID.Type()]
		if err := validateConnectorUse(factory, connID, expPipelines, recPipelines); err != nil {
			return err
		}
	}

	return nil
}

// Given the set of pipelines in which a connector is used as an exporter and receiver,
// we need to validate that all pipelines are using the connector in a supported way.
//
// For example, consider the following config, in which both logs and metrics are forwarded:
//
//	pipelines:
//	  logs/in:
//	    receivers: [otlp]
//	    exporters: [forward]
//	  logs/out:
//	    receivers: [forward]
//	    exporters: [otlp]
//	  metrics/in:
//	    receivers: [otlp]
//	    exporters: [forward]
//	  metrics/out:
//	    receivers: [forward]
//	    exporters: [otlp]
//
// When validating this configuration, we look at each use of the connector and confirm that there
// is a valid corresponding use in another pipeline:
// - As an exporter in logs/in: Valid, because it has a corresponding use as a receiver in logs/out
// - As a receiver in logs/out: Valid, because it has a corresponding use as an exporter in logs/in
// - As an exporter in metrics/in: Valid, because it has a corresponding use as a receiver in metrics/out
// - As a receiver in metrics/out: Valid, because it has a corresponding use as an exporter in metrics/in
// We conclude that it is used correctly, because no uses are unconnected via a supported combination.
//
// Now consider the following config, in which we validation should fail:
//
//	pipelines:
//	  traces/in:
//	    receivers: [otlp]
//	    exporters: [forward]
//	  traces/out:
//	    receivers: [forward]
//	    exporters: [otlp]
//	  metrics/in:
//	    receivers: [otlp]
//	    exporters: [forward]
//
// When validating this configuration, we look at each use of the connector and find:
// - As an exporter in traces/in: Valid, because it has a corresponding use as a receiver in traces/out
// - As a receiver in traces/out: Valid, because it has a corresponding use as an exporter in traces/in
// - As an exporter in metrics/in: *Invalid*, because it has a corresponding use as a receiver in a metrics pipeline
// We conclude that it is used incorrectly, because at least one use is unconnected via a supported combination.
func validateConnectorUse(factory connector.Factory, connID component.ID, expPipelines, recPipelines []component.ID) error {
	expTypes := make(map[component.DataType]bool)
	for _, pipelineID := range expPipelines {
		// The presence of each key indicates how the connector is used as an exporter.
		// The value is initially set to false. Later we will set the value to true *if* we
		// confirm that there is a supported corresponding use as a receiver.
		expTypes[pipelineID.Type()] = false
	}
	recTypes := make(map[component.DataType]bool)
	for _, pipelineID := range recPipelines {
		// The presence of each key indicates how the connector is used as a receiver.
		// The value is initially set to false. Later we will set the value to true *if* we
		// confirm that there is a supported corresponding use as an exporter.
		recTypes[pipelineID.Type()] = false
	}

	for expType := range expTypes {
		for recType := range recTypes {
			if factory.Stability(expType, recType) != component.StabilityLevelUndefined {
				expTypes[expType] = true
				recTypes[recType] = true
			}
		}
	}

	for expType, supportedUse := range expTypes {
		if !supportedUse {
			return fmt.Errorf("connectors::%s: is used as exporter in %q pipeline but is not used in supported receiver pipeline", connID, expType)
		}
	}
	for recType, supportedUse := range recTypes {
		if !supportedUse {
			return fmt.Errorf("connectors::%s: is used as receiver in %q pipeline but is not used in supported exporter pipeline", connID, recType)
		}
	}

	return nil
}
