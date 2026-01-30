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

func TestCategorizeConfigChange(t *testing.T) {
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
		want        ConfigChangeType
	}{
		{
			name:        "identical_configs",
			oldCfg:      baseConfig(),
			newCfg:      baseConfig(),
			isConnector: neverConnector,
			// categorizeConfigChange returns PartialReload when full reload isn't needed.
			// Graph.Reload handles detecting that no actual changes occurred.
			want: ConfigChangePartialReload,
		},
		{
			name:   "receiver_config_changed",
			oldCfg: baseConfig(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Receivers[component.MustNewID("otlp")] = "changed"
				return c
			}(),
			isConnector: neverConnector,
			want:        ConfigChangePartialReload,
		},
		{
			name:   "receiver_added_to_config_map",
			oldCfg: baseConfig(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Receivers[component.MustNewID("jaeger")] = struct{}{}
				return c
			}(),
			isConnector: neverConnector,
			want:        ConfigChangePartialReload,
		},
		{
			name:   "receiver_added_to_pipeline",
			oldCfg: baseConfig(),
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
			want:        ConfigChangePartialReload,
		},
		{
			name:   "processor_config_changed",
			oldCfg: baseConfig(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Processors[component.MustNewID("batch")] = "changed"
				return c
			}(),
			isConnector: neverConnector,
			want:        ConfigChangePartialReload,
		},
		{
			name:   "processor_added_to_config_map",
			oldCfg: baseConfig(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Processors[component.MustNewIDWithName("batch", "2")] = struct{}{}
				return c
			}(),
			isConnector: neverConnector,
			want:        ConfigChangePartialReload,
		},
		{
			name:   "processor_added_to_pipeline",
			oldCfg: baseConfig(),
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
			want:        ConfigChangePartialReload,
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
			want:        ConfigChangePartialReload,
		},
		{
			name:   "exporter_config_changed",
			oldCfg: baseConfig(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Exporters[component.MustNewID("otlp")] = "changed"
				return c
			}(),
			isConnector: neverConnector,
			// Exporter config changes can be handled by partial reload.
			want: ConfigChangePartialReload,
		},
		{
			name:   "pipeline_added",
			oldCfg: baseConfig(),
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
			// Pipeline without connectors can be added via partial reload.
			want: ConfigChangePartialReload,
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
			// Pipeline without connectors can be removed via partial reload.
			want: ConfigChangePartialReload,
		},
		{
			name:   "pipeline_added_with_connector_as_receiver",
			oldCfg: baseConfig(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Connectors = map[component.ID]component.Config{
					connectorID: struct{}{},
				}
				c.Service.Pipelines[pipeline.NewID(pipeline.SignalMetrics)] = &pipelines.PipelineConfig{
					Receivers:  []component.ID{connectorID}, // connector as receiver
					Processors: []component.ID{component.MustNewID("batch")},
					Exporters:  []component.ID{component.MustNewID("otlp")},
				}
				return c
			}(),
			isConnector: isForwardConnector,
			// Pipeline with connector can now be handled by partial reload.
			want: ConfigChangePartialReload,
		},
		{
			name:   "pipeline_added_with_connector_as_exporter",
			oldCfg: baseConfig(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Connectors = map[component.ID]component.Config{
					connectorID: struct{}{},
				}
				c.Service.Pipelines[pipeline.NewID(pipeline.SignalMetrics)] = &pipelines.PipelineConfig{
					Receivers:  []component.ID{component.MustNewID("otlp")},
					Processors: []component.ID{component.MustNewID("batch")},
					Exporters:  []component.ID{connectorID}, // connector as exporter
				}
				return c
			}(),
			isConnector: isForwardConnector,
			// Pipeline with connector can now be handled by partial reload.
			want: ConfigChangePartialReload,
		},
		{
			name: "pipeline_removed_with_connector",
			oldCfg: func() *Config {
				c := baseConfig()
				c.Connectors = map[component.ID]component.Config{
					connectorID: struct{}{},
				}
				c.Service.Pipelines[pipeline.NewID(pipeline.SignalMetrics)] = &pipelines.PipelineConfig{
					Receivers:  []component.ID{connectorID}, // connector as receiver
					Processors: []component.ID{component.MustNewID("batch")},
					Exporters:  []component.ID{component.MustNewID("otlp")},
				}
				return c
			}(),
			newCfg:      baseConfig(),
			isConnector: isForwardConnector,
			// Pipeline with connector can now be handled by partial reload.
			want: ConfigChangePartialReload,
		},
		{
			name:   "exporter_config_map_changed",
			oldCfg: baseConfig(),
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
			// Adding exporter to config and pipeline can be handled by partial reload.
			want: ConfigChangePartialReload,
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
			// Changing pipeline exporter list can be handled by partial reload.
			want: ConfigChangePartialReload,
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
			// Connector config changes can be handled by partial reload.
			want: ConfigChangePartialReload,
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
			// Adding connector as receiver can be handled by partial reload.
			want: ConfigChangePartialReload,
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
			want:        ConfigChangeFullReload,
		},
		{
			name:   "extensions_list_changed",
			oldCfg: baseConfig(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Extensions = map[component.ID]component.Config{
					component.MustNewID("health"): struct{}{},
				}
				c.Service.Extensions = []component.ID{component.MustNewID("health")}
				return c
			}(),
			isConnector: neverConnector,
			want:        ConfigChangeFullReload,
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
			want:        ConfigChangePartialReload,
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
			want:        ConfigChangeFullReload,
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
			// Removing metrics and adding logs pipelines (both without connectors) can be handled by partial reload.
			want: ConfigChangePartialReload,
		},
		{
			name:   "receiver_and_processor_both_changed",
			oldCfg: baseConfig(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Receivers[component.MustNewID("otlp")] = "changed"
				c.Processors[component.MustNewID("batch")] = "changed"
				return c
			}(),
			isConnector: neverConnector,
			want:        ConfigChangePartialReload,
		},
		{
			// Test case where receiver config map is unchanged but pipeline receiver list changed.
			// This covers the inner loop in receiver detection.
			name: "receiver_list_changed_config_unchanged",
			oldCfg: func() *Config {
				c := baseConfig()
				// Add a second receiver to the config map but not to the pipeline.
				c.Receivers[component.MustNewIDWithName("otlp", "2")] = struct{}{}
				return c
			}(),
			newCfg: func() *Config {
				c := baseConfig()
				// Same config map, but now the second receiver is in the pipeline.
				c.Receivers[component.MustNewIDWithName("otlp", "2")] = struct{}{}
				c.Service.Pipelines[pipeline.NewID(pipeline.SignalTraces)].Receivers = []component.ID{
					component.MustNewID("otlp"),
					component.MustNewIDWithName("otlp", "2"),
				}
				return c
			}(),
			isConnector: neverConnector,
			want:        ConfigChangePartialReload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := categorizeConfigChange(tt.oldCfg, tt.newCfg, tt.isConnector)
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
