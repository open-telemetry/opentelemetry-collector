// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service // import "go.opentelemetry.io/collector/service"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/telemetry"
)

var (
	errMissingServicePipelines         = errors.New("service must have at least one pipeline")
	errMissingServicePipelineReceivers = errors.New("must have at least one receiver")
	errMissingServicePipelineExporters = errors.New("must have at least one exporter")
)

// Config defines the configurable components of the Service.
type Config struct {
}

// Validate returns an error if the config is invalid.
//
// This function performs basic validation of configuration. There may be more subtle
// invalid cases that we currently don't check for but which we may want to add in
// the future (e.g. disallowing receiving and exporting on the same endpoint).
func (cfg *Config) Validate() error {
	// Currently, there is no default receiver enabled.
	// The configuration must specify at least one receiver to be valid.
	if len(cfg.Receivers) == 0 {
		return errMissingReceivers
	}

	// Validate the receiver configuration.
	for recvID, recvCfg := range cfg.Receivers {
		if err := recvCfg.Validate(); err != nil {
			return fmt.Errorf("receiver %q has invalid configuration: %w", recvID, err)
		}
	}

	// Currently, there is no default exporter enabled.
	// The configuration must specify at least one exporter to be valid.
	if len(cfg.Exporters) == 0 {
		return errMissingExporters
	}

	// Validate the exporter configuration.
	for expID, expCfg := range cfg.Exporters {
		if err := expCfg.Validate(); err != nil {
			return fmt.Errorf("exporter %q has invalid configuration: %w", expID, err)
		}
	}

	// Validate the processor configuration.
	for procID, procCfg := range cfg.Processors {
		if err := procCfg.Validate(); err != nil {
			return fmt.Errorf("processor %q has invalid configuration: %w", procID, err)
		}
	}

	// Validate the extension configuration.
	for extID, extCfg := range cfg.Extensions {
		if err := extCfg.Validate(); err != nil {
			return fmt.Errorf("extension %q has invalid configuration: %w", extID, err)
		}
	}

	return cfg.validateService()
}

func (cfg *Config) validateService() error {
	// Check that all enabled extensions in the service are configured.
	for _, ref := range cfg.Service.Extensions {
		// Check that the name referenced in the Service extensions exists in the top-level extensions.
		if cfg.Extensions[ref] == nil {
			return fmt.Errorf("service references extension %q which does not exist", ref)
		}
	}

	// Must have at least one pipeline.
	if len(cfg.Service.Pipelines) == 0 {
		return errMissingServicePipelines
	}

	// Check that all pipelines have at least one receiver and one exporter, and they reference
	// only configured components.
	for pipelineID, pipeline := range cfg.Service.Pipelines {
		if pipelineID.Type() != component.DataTypeTraces && pipelineID.Type() != component.DataTypeMetrics && pipelineID.Type() != component.DataTypeLogs {
			return fmt.Errorf("unknown pipeline datatype %q for %v", pipelineID.Type(), pipelineID)
		}

		// Validate pipeline has at least one receiver.
		if len(pipeline.Receivers) == 0 {
			return fmt.Errorf("pipeline %q must have at least one receiver", pipelineID)
		}

		// Validate pipeline receiver name references.
		for _, ref := range pipeline.Receivers {
			// Check that the name referenced in the pipeline's receivers exists in the top-level receivers.
			if cfg.Receivers[ref] == nil {
				return fmt.Errorf("pipeline %q references receiver %q which does not exist", pipelineID, ref)
			}
		}

		// Validate pipeline processor name references.
		for _, ref := range pipeline.Processors {
			// Check that the name referenced in the pipeline's processors exists in the top-level processors.
			if cfg.Processors[ref] == nil {
				return fmt.Errorf("pipeline %q references processor %q which does not exist", pipelineID, ref)
			}
		}

		// Validate pipeline has at least one exporter.
		if len(pipeline.Exporters) == 0 {
			return fmt.Errorf("pipeline %q must have at least one exporter", pipelineID)
		}

		// Validate pipeline exporter name references.
		for _, ref := range pipeline.Exporters {
			// Check that the name referenced in the pipeline's Exporters exists in the top-level Exporters.
			if cfg.Exporters[ref] == nil {
				return fmt.Errorf("pipeline %q references exporter %q which does not exist", pipelineID, ref)
			}
		}

		if err := cfg.Service.Telemetry.Validate(); err != nil {
			fmt.Printf("telemetry config validation failed, %v\n", err)
		}
	}
	return nil
}

func (cfg *Config) DryRunValidate() {
	// Currently, there is no default receiver enabled.
	// The configuration must specify at least one receiver to be valid.
	if len(cfg.Receivers) == 0 {
		fmt.Printf("**..%v\n", errMissingReceivers)
	} else {
		for recvID, recvCfg := range cfg.Receivers {
			if err := recvCfg.Validate(); err != nil {
				fmt.Printf("**..receiver %q has invalid configuration: %v\n", recvID, err)
			}
		}
	}

	// Currently, there is no default exporter enabled.
	// The configuration must specify at least one exporter to be valid.
	if len(cfg.Exporters) == 0 {
		fmt.Printf("**..%v\n", errMissingExporters)
	} else {
		for expID, expCfg := range cfg.Exporters {
			if err := expCfg.Validate(); err != nil {
				fmt.Printf("**..exporter %q has invalid configuration: %v\n", expID, err)
			}
		}
	}

	// Validate the processor configuration.
	for procID, procCfg := range cfg.Processors {
		if err := procCfg.Validate(); err != nil {
			fmt.Printf("**..processor %q has invalid configuration: %v\n", procID, err)
		}
	}

	// Validate the extension configuration.
	for extID, extCfg := range cfg.Extensions {
		if err := extCfg.Validate(); err != nil {
			fmt.Printf("**..extension %q has invalid configuration: %v\n", extID, err)
		}
	}
	cfg.dryRunValidateService()
}

func (cfg *Config) dryRunValidateService() {

	for _, ref := range cfg.Service.Extensions {
		// Check that the name referenced in the Service extensions exists in the top-level extensions.
		if cfg.Extensions[ref] == nil {
			fmt.Printf("**..service references extension %q which does not exist\n", ref)
		}
	}

	// Must have at least one pipeline.
	if len(cfg.Service.Pipelines) == 0 {
		fmt.Printf("**..%v\n", errMissingServicePipelines)
	} else {
		for pipelineID, pipeline := range cfg.Service.Pipelines {
			if pipelineID.Type() != component.DataTypeTraces && pipelineID.Type() != component.DataTypeMetrics && pipelineID.Type() != component.DataTypeLogs {
				fmt.Printf("unknown pipeline datatype %q for %v", pipelineID.Type(), pipelineID)
			}

			// Validate pipeline has at least one receiver.
			if len(pipeline.Receivers) == 0 {
				fmt.Printf("**..pipeline %q must have at least one receiver\n", pipelineID)
			} else {
				for _, ref := range pipeline.Receivers {
					// Check that the name referenced in the pipeline's receivers exists in the top-level receivers.
					if cfg.Receivers[ref] == nil {
						fmt.Printf("**..pipeline %q references receiver %q which does not exist\n", pipelineID, ref)
					}
				}
			}
			for _, ref := range pipeline.Processors {
				// Check that the name referenced in the pipeline's processors exists in the top-level processors.
				if cfg.Processors[ref] == nil {
					fmt.Printf("**..pipeline %q references processor %q which does not exist\n", pipelineID, ref)
				}
			}

			// Validate pipeline has at least one exporter.
			if len(pipeline.Exporters) == 0 {
				fmt.Printf("**..pipeline %q must have at least one exporter\n", pipelineID)
			} else {
				for _, ref := range pipeline.Exporters {
					// Check that the name referenced in the pipeline's Exporters exists in the top-level Exporters.
					if cfg.Exporters[ref] == nil {
						fmt.Printf("**..pipeline %q references exporter %q which does not exist\n", pipelineID, ref)
					}
				}
			}
		}
	}
}

// ConfigService defines the configurable components of the service.
type ConfigService struct {
	// Telemetry is the configuration for collector's own telemetry.
	Telemetry telemetry.Config `mapstructure:"telemetry"`

	// Extensions are the ordered list of extensions configured for the service.
	Extensions []component.ID `mapstructure:"extensions"`

	// Pipelines are the set of data pipelines configured for the service.
	Pipelines map[component.ID]*PipelineConfig `mapstructure:"pipelines"`
}

func (cfg *Config) Validate() error {
	// Must have at least one pipeline.
	if len(cfg.Pipelines) == 0 {
		return errMissingServicePipelines
	}

	// Check that all pipelines have at least one receiver and one exporter, and they reference
	// only configured components.
	for pipelineID, pipeline := range cfg.Pipelines {
		if pipelineID.Type() != component.DataTypeTraces && pipelineID.Type() != component.DataTypeMetrics && pipelineID.Type() != component.DataTypeLogs {
			return fmt.Errorf("service::pipeline::%s: unknown datatype %q", pipelineID, pipelineID.Type())
		}

		// Validate pipeline has at least one receiver.
		if err := pipeline.Validate(); err != nil {
			return fmt.Errorf("service::pipeline::%s: %w", pipelineID, err)
		}
	}

	if err := cfg.Telemetry.Validate(); err != nil {
		fmt.Printf("service::telemetry config validation failed, %v\n", err)
	}

	return nil
}

// PipelineConfig defines the configuration of a Pipeline.
type PipelineConfig struct {
	Receivers  []component.ID `mapstructure:"receivers"`
	Processors []component.ID `mapstructure:"processors"`
	Exporters  []component.ID `mapstructure:"exporters"`
}

func (cfg *PipelineConfig) Validate() error {
	// Validate pipeline has at least one receiver.
	if len(cfg.Receivers) == 0 {
		return errMissingServicePipelineReceivers
	}

	// Validate pipeline has at least one exporter.
	if len(cfg.Exporters) == 0 {
		return errMissingServicePipelineExporters
	}

	// Validate no processors are duplicated within a pipeline.
	procSet := make(map[component.ID]struct{}, len(cfg.Processors))
	for _, ref := range cfg.Processors {
		// Ensure no processors are duplicated within the pipeline
		if _, exists := procSet[ref]; exists {
			return fmt.Errorf("references processor %q multiple times", ref)
		}
		procSet[ref] = struct{}{}
	}

	return nil
}
