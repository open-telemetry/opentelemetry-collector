// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pipelines // import "go.opentelemetry.io/collector/service/pipelines"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/service/internal/metadata"
)

var (
	errMissingServicePipelines         = errors.New("service must have at least one pipeline")
	errMissingServicePipelineReceivers = errors.New("must have at least one receiver")
	errMissingServicePipelineExporters = errors.New("must have at least one exporter")

	AllowNoPipelines = metadata.ServiceAllowNoPipelinesFeatureGate
)

// Config defines the configurable settings for service telemetry.
type Config map[pipeline.ID]*PipelineConfig

func (cfg Config) Validate() error {
	// Must have at least one pipeline unless explicitly disabled.
	if !metadata.ServiceAllowNoPipelinesFeatureGate.IsEnabled() && len(cfg) == 0 {
		return errMissingServicePipelines
	}

	if !metadata.ServiceProfilesSupportFeatureGate.IsEnabled() {
		// Check that all pipelines have at least one receiver and one exporter, and they reference
		// only configured components.
		for pipelineID := range cfg {
			if pipelineID.Signal() == xpipeline.SignalProfiles {
				return fmt.Errorf(
					"pipeline %q: profiling signal support is at alpha level, gated under the %q feature gate",
					pipelineID.String(),
					metadata.ServiceProfilesSupportFeatureGate.ID(),
				)
			}
		}
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
