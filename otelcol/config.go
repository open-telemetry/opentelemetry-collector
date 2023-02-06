// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/otelcol/internal/sharedgate"
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
func (cfg *Config) Validate() error {
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

	if len(cfg.Connectors) != 0 && !sharedgate.ConnectorsFeatureGate.IsEnabled() {
		return fmt.Errorf("connectors require feature gate: %s", sharedgate.ConnectorsFeatureGate.ID())
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

	// Keep track of whether connectors are used as receivers and exporters
	connectorsAsReceivers := make(map[component.ID]struct{}, len(cfg.Connectors))
	connectorsAsExporters := make(map[component.ID]struct{}, len(cfg.Connectors))

	// Check that all pipelines reference only configured components.
	for pipelineID, pipeline := range cfg.Service.Pipelines {
		// Validate pipeline receiver name references.
		for _, ref := range pipeline.Receivers {
			// Check that the name referenced in the pipeline's receivers exists in the top-level receivers.
			if _, ok := cfg.Receivers[ref]; ok {
				continue
			}

			if _, ok := cfg.Connectors[ref]; ok {
				connectorsAsReceivers[ref] = struct{}{}
				continue
			}
			return fmt.Errorf("service::pipeline::%s: references receiver %q which is not configured", pipelineID, ref)
		}

		// Validate pipeline processor name references.
		for _, ref := range pipeline.Processors {
			// Check that the name referenced in the pipeline's processors exists in the top-level processors.
			if cfg.Processors[ref] == nil {
				return fmt.Errorf("service::pipeline::%s: references processor %q which is not configured", pipelineID, ref)
			}
		}

		// Validate pipeline exporter name references.
		for _, ref := range pipeline.Exporters {
			// Check that the name referenced in the pipeline's Exporters exists in the top-level Exporters.
			if _, ok := cfg.Exporters[ref]; ok {
				continue
			}
			if _, ok := cfg.Connectors[ref]; ok {
				connectorsAsExporters[ref] = struct{}{}
				continue
			}
			return fmt.Errorf("service::pipeline::%s: references exporter %q which is not configured", pipelineID, ref)
		}
	}

	// Validate that connectors are used as both receiver and exporter
	for connID := range cfg.Connectors {
		_, recOK := connectorsAsReceivers[connID]
		_, expOK := connectorsAsExporters[connID]
		if recOK && !expOK {
			return fmt.Errorf("connectors::%s: must be used as both receiver and exporter but is not used as exporter", connID)
		}
		if !recOK && expOK {
			return fmt.Errorf("connectors::%s: must be used as both receiver and exporter but is not used as receiver", connID)
		}
	}

	return nil
}
