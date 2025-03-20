// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pipelines // import "go.opentelemetry.io/collector/service/pipelines"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

var (
	errMissingServicePipelines         = errors.New("service must have at least one pipeline")
	errMissingServicePipelineReceivers = errors.New("must have at least one receiver")
	errMissingServicePipelineExporters = errors.New("must have at least one exporter")

	serviceProfileSupportGateID = "service.profilesSupport"
	serviceProfileSupportGate   = featuregate.GlobalRegistry().MustRegister(
		serviceProfileSupportGateID,
		featuregate.StageAlpha,
		featuregate.WithRegisterFromVersion("v0.112.0"),
		featuregate.WithRegisterDescription("Controls whether profiles support can be enabled"),
	)
	AllowNoPipelines = featuregate.GlobalRegistry().MustRegister(
		"service.AllowNoPipelines",
		featuregate.StageAlpha,
		featuregate.WithRegisterFromVersion("v0.122.0"),
		featuregate.WithRegisterDescription("Allow starting the Collector without starting any pipelines."),
	)
)

// Config defines the configurable settings for service telemetry.
type Config map[pipeline.ID]*PipelineConfig

func (cfg Config) Validate() error {
	// Must have at least one pipeline unless explicitly disabled.
	if !AllowNoPipelines.IsEnabled() && len(cfg) == 0 {
		return errMissingServicePipelines
	}

	// Check that all pipelines have at least one receiver and one exporter, and they reference
	// only configured components.
	for pipelineID := range cfg {
		switch pipelineID.Signal() {
		case pipeline.SignalTraces, pipeline.SignalMetrics, pipeline.SignalLogs:
			// Continue
		case xpipeline.SignalProfiles:
			if !serviceProfileSupportGate.IsEnabled() {
				return fmt.Errorf(
					"pipeline %q: profiling signal support is at alpha level, gated under the %q feature gate",
					pipelineID.String(),
					serviceProfileSupportGateID,
				)
			}
		default:
			return fmt.Errorf("pipeline %q: unknown signal %q", pipelineID.String(), pipelineID.Signal())
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
