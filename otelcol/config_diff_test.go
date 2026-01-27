// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/pipelines"
)

func TestReceiversOnlyChange(t *testing.T) {
	connectorID := component.MustNewID("forward")
	neverConnector := func(component.ID) bool { return false }
	isForwardConnector := func(id component.ID) bool { return id == connectorID }

	baseConfig := func() *Config {
		return &Config{
			Receivers: map[component.ID]component.Config{
				component.MustNewID("otlp"): struct{}{},
			},
			Processors: map[component.ID]component.Config{
				component.MustNewID("batch"): struct{}{},
			},
			Exporters: map[component.ID]component.Config{
				component.MustNewID("otlp"): struct{}{},
			},
			Extensions: map[component.ID]component.Config{},
			Connectors: map[component.ID]component.Config{},
			Service: service.Config{
				Pipelines: pipelines.Config{
					pipeline.NewID(pipeline.SignalTraces): {
						Receivers:  []component.ID{component.MustNewID("otlp")},
						Processors: []component.ID{component.MustNewID("batch")},
						Exporters:  []component.ID{component.MustNewID("otlp")},
					},
				},
			},
		}
	}

	tests := []struct {
		name        string
		oldCfg      *Config
		newCfg      *Config
		isConnector func(component.ID) bool
		want        bool
	}{
		{
			name:        "identical_configs",
			oldCfg:      baseConfig(),
			newCfg:      baseConfig(),
			isConnector: neverConnector,
			want:        true,
		},
		{
			name: "receiver_config_changed",
			oldCfg:  baseConfig(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Receivers[component.MustNewID("otlp")] = "changed"
				return c
			}(),
			isConnector: neverConnector,
			want:        true,
		},
		{
			name: "receiver_added_to_config_map",
			oldCfg:  baseConfig(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Receivers[component.MustNewID("jaeger")] = struct{}{}
				return c
			}(),
			isConnector: neverConnector,
			want:        true,
		},
		{
			name: "receiver_added_to_pipeline",
			oldCfg:  baseConfig(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Receivers[component.MustNewID("jaeger")] = struct{}{}
				c.Service.Pipelines[pipeline.NewID(pipeline.SignalTraces)].Receivers = append(
					c.Service.Pipelines[pipeline.NewID(pipeline.SignalTraces)].Receivers,
					component.MustNewID("jaeger"),
				)
				return c
			}(),
			isConnector: neverConnector,
			want:        true,
		},
		{
			name: "processor_config_changed",
			oldCfg:  baseConfig(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Processors[component.MustNewID("batch")] = "changed"
				return c
			}(),
			isConnector: neverConnector,
			want:        false,
		},
		{
			name: "exporter_config_changed",
			oldCfg:  baseConfig(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Exporters[component.MustNewID("otlp")] = "changed"
				return c
			}(),
			isConnector: neverConnector,
			want:        false,
		},
		{
			name: "pipeline_added",
			oldCfg:  baseConfig(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Service.Pipelines[pipeline.NewID(pipeline.SignalMetrics)] = &pipelines.PipelineConfig{
					Receivers:  []component.ID{component.MustNewID("otlp")},
					Processors: []component.ID{component.MustNewID("batch")},
					Exporters:  []component.ID{component.MustNewID("otlp")},
				}
				return c
			}(),
			isConnector: neverConnector,
			want:        false,
		},
		{
			name: "pipeline_removed",
			oldCfg: func() *Config {
				c := baseConfig()
				c.Service.Pipelines[pipeline.NewID(pipeline.SignalMetrics)] = &pipelines.PipelineConfig{
					Receivers:  []component.ID{component.MustNewID("otlp")},
					Processors: []component.ID{component.MustNewID("batch")},
					Exporters:  []component.ID{component.MustNewID("otlp")},
				}
				return c
			}(),
			newCfg:      baseConfig(),
			isConnector: neverConnector,
			want:        false,
		},
		{
			name: "processor_config_map_changed",
			oldCfg:  baseConfig(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Processors[component.MustNewIDWithName("batch", "2")] = struct{}{}
				c.Service.Pipelines[pipeline.NewID(pipeline.SignalTraces)].Processors = []component.ID{
					component.MustNewID("batch"),
					component.MustNewIDWithName("batch", "2"),
				}
				return c
			}(),
			isConnector: neverConnector,
			want:        false,
		},
		{
			name: "pipeline_processors_list_changed",
			oldCfg: func() *Config {
				c := baseConfig()
				c.Processors[component.MustNewIDWithName("batch", "2")] = struct{}{}
				return c
			}(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Processors[component.MustNewIDWithName("batch", "2")] = struct{}{}
				c.Service.Pipelines[pipeline.NewID(pipeline.SignalTraces)].Processors = []component.ID{
					component.MustNewID("batch"),
					component.MustNewIDWithName("batch", "2"),
				}
				return c
			}(),
			isConnector: neverConnector,
			want:        false,
		},
		{
			name: "exporter_config_map_changed",
			oldCfg:  baseConfig(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Exporters[component.MustNewIDWithName("otlp", "2")] = struct{}{}
				c.Service.Pipelines[pipeline.NewID(pipeline.SignalTraces)].Exporters = []component.ID{
					component.MustNewID("otlp"),
					component.MustNewIDWithName("otlp", "2"),
				}
				return c
			}(),
			isConnector: neverConnector,
			want:        false,
		},
		{
			name: "pipeline_exporters_list_changed",
			oldCfg: func() *Config {
				c := baseConfig()
				c.Exporters[component.MustNewIDWithName("otlp", "2")] = struct{}{}
				return c
			}(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Exporters[component.MustNewIDWithName("otlp", "2")] = struct{}{}
				c.Service.Pipelines[pipeline.NewID(pipeline.SignalTraces)].Exporters = []component.ID{
					component.MustNewID("otlp"),
					component.MustNewIDWithName("otlp", "2"),
				}
				return c
			}(),
			isConnector: neverConnector,
			want:        false,
		},
		{
			name: "connector_config_changed",
			oldCfg: func() *Config {
				c := baseConfig()
				c.Connectors = map[component.ID]component.Config{
					connectorID: struct{}{},
				}
				return c
			}(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Connectors = map[component.ID]component.Config{
					connectorID: "changed",
				}
				return c
			}(),
			isConnector: isForwardConnector,
			want:        false,
		},
		{
			name: "connector_as_receiver_changed",
			oldCfg: func() *Config {
				c := baseConfig()
				c.Connectors = map[component.ID]component.Config{
					connectorID: struct{}{},
				}
				return c
			}(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Connectors = map[component.ID]component.Config{
					connectorID: struct{}{},
				}
				c.Service.Pipelines[pipeline.NewID(pipeline.SignalTraces)].Receivers = append(
					c.Service.Pipelines[pipeline.NewID(pipeline.SignalTraces)].Receivers,
					connectorID,
				)
				return c
			}(),
			isConnector: isForwardConnector,
			want:        false,
		},
		{
			name: "extension_config_changed",
			oldCfg: func() *Config {
				c := baseConfig()
				c.Extensions = map[component.ID]component.Config{
					component.MustNewID("health"): struct{}{},
				}
				return c
			}(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Extensions = map[component.ID]component.Config{
					component.MustNewID("health"): "changed",
				}
				return c
			}(),
			isConnector: neverConnector,
			want:        false,
		},
		{
			name: "extensions_list_changed",
			oldCfg:  baseConfig(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Extensions = map[component.ID]component.Config{
					component.MustNewID("health"): struct{}{},
				}
				c.Service.Extensions = []component.ID{component.MustNewID("health")}
				return c
			}(),
			isConnector: neverConnector,
			want:        false,
		},
		{
			name: "receiver_removed_from_config_map",
			oldCfg: func() *Config {
				c := baseConfig()
				c.Receivers[component.MustNewID("jaeger")] = struct{}{}
				return c
			}(),
			newCfg:      baseConfig(),
			isConnector: neverConnector,
			want:        true,
		},
		{
			name:   "telemetry_changed",
			oldCfg: baseConfig(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Service.Telemetry = "changed"
				return c
			}(),
			isConnector: neverConnector,
			want:        false,
		},
		{
			name: "pipeline_id_replaced",
			oldCfg: func() *Config {
				c := baseConfig()
				c.Service.Pipelines[pipeline.NewID(pipeline.SignalMetrics)] = &pipelines.PipelineConfig{
					Receivers:  []component.ID{component.MustNewID("otlp")},
					Processors: []component.ID{component.MustNewID("batch")},
					Exporters:  []component.ID{component.MustNewID("otlp")},
				}
				return c
			}(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Service.Pipelines[pipeline.NewID(pipeline.SignalLogs)] = &pipelines.PipelineConfig{
					Receivers:  []component.ID{component.MustNewID("otlp")},
					Processors: []component.ID{component.MustNewID("batch")},
					Exporters:  []component.ID{component.MustNewID("otlp")},
				}
				return c
			}(),
			isConnector: neverConnector,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := receiversOnlyChange(tt.oldCfg, tt.newCfg, tt.isConnector)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsConnectorID(t *testing.T) {
	connID := component.MustNewID("forward")
	receiverID := component.MustNewID("otlp")

	connectors := map[component.ID]component.Config{
		connID: struct{}{},
	}
	pred := isConnectorID(connectors)

	assert.True(t, pred(connID))
	assert.False(t, pred(receiverID))
}
