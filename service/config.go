// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service // import "go.opentelemetry.io/collector/service"

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/pipelines"
	"go.opentelemetry.io/collector/service/telemetry"
)

var PipelineLessMode = featuregate.GlobalRegistry().MustRegister(
	"service.pipelineless",
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription("controls whether the 'pipelines' are used in the service config"))

// Config defines the configurable components of the Service.
type Config struct {
	// Telemetry is the configuration for collector's own telemetry.
	Telemetry telemetry.Config `mapstructure:"telemetry"`

	// Extensions are the ordered list of extensions configured for the service.
	Extensions extensions.Config `mapstructure:"extensions"`

	// Pipelines are the set of data pipelines configured for the service.
	Pipelines pipelines.Config `mapstructure:"pipelines"`

	// Alternate way to specify relationships between components
	DataFlow DataFlowConfig `mapstructure:"dataflow"`
}

type DataFlowConfig struct {
	Traces  map[component.ID][]component.ID `mapstructure:"traces"`
	Metrics map[component.ID][]component.ID `mapstructure:"metrics"`
	Logs    map[component.ID][]component.ID `mapstructure:"logs"`
}

func (cfg *Config) ConvertToPipelines() {
	cfg.Pipelines = make(map[component.ID]*pipelines.PipelineConfig)

	for from, to := range cfg.DataFlow.Traces {
		pipelineID := component.NewIDWithName(component.DataTypeTraces, fmt.Sprintf("from-%s", from.String()))
		cfg.Pipelines[pipelineID] = &pipelines.PipelineConfig{
			Receivers: []component.ID{from},
			Exporters: to,
		}
	}
	for from, to := range cfg.DataFlow.Metrics {
		pipelineID := component.NewIDWithName(component.DataTypeMetrics, fmt.Sprintf("from-%s", from.String()))
		cfg.Pipelines[pipelineID] = &pipelines.PipelineConfig{
			Receivers: []component.ID{from},
			Exporters: to,
		}
	}
	for from, to := range cfg.DataFlow.Logs {
		pipelineID := component.NewIDWithName(component.DataTypeLogs, fmt.Sprintf("from-%s", from.String()))
		cfg.Pipelines[pipelineID] = &pipelines.PipelineConfig{
			Receivers: []component.ID{from},
			Exporters: to,
		}
	}
}

func (cfg *Config) Validate() error {
	if !PipelineLessMode.IsEnabled() {
		if len(cfg.DataFlow.Traces) > 0 {
			return fmt.Errorf("service::traces config is not allowed when pipelineless mode is disabled")
		}
		if len(cfg.DataFlow.Metrics) > 0 {
			return fmt.Errorf("service::metrics config is not allowed when pipelineless mode is disabled")
		}
		if len(cfg.DataFlow.Logs) > 0 {
			return fmt.Errorf("service::logs config is not allowed when pipelineless mode is disabled")
		}
	}

	if err := cfg.Pipelines.Validate(); err != nil {
		return fmt.Errorf("service::pipelines config validation failed: %w", err)
	}

	if err := cfg.Telemetry.Validate(); err != nil {
		fmt.Printf("service::telemetry config validation failed: %v\n", err)
	}

	return nil
}
