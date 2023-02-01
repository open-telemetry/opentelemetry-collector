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
