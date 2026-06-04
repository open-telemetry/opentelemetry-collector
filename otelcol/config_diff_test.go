// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/pipelines"
)

// opaqueConfig is a minimal component config holding a secret, used to verify
// that the diff distinguishes configs that differ only in an opaque field.
type opaqueConfig struct {
	Token configopaque.String
}

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
			name:   "receiver_config_changed",
			oldCfg: baseConfig(),
			newCfg: func() *Config {
				c := baseConfig()
				c.Receivers[component.MustNewID("otlp")] = "changed"
				return c
			}(),
			isConnector: neverConnector,
			want:        true,
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
			want:        true,
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
			want:        true,
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
			want:        false,
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
			want:        false,
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
			name:   "processor_config_map_changed",
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

// TestReceiversOnlyChangeOpaqueSecrets verifies that the diff observes changes
// to configopaque.String fields. This is the reason receiversOnlyChange uses
// reflect.DeepEqual rather than a serialization-based comparison: opaque
// strings marshal to "[REDACTED]", so a marshal-based diff would treat configs
// with different secrets as identical and silently skip a necessary reload.
func TestReceiversOnlyChangeOpaqueSecrets(t *testing.T) {
	// Sanity check: the two secrets are indistinguishable once marshaled, so a
	// serialization-based diff would wrongly consider them equal.
	old, err := configopaque.String("secret1").MarshalText()
	require.NoError(t, err)
	updated, err := configopaque.String("secret2").MarshalText()
	require.NoError(t, err)
	require.Equal(t, string(old), string(updated), "opaque secrets should marshal identically")

	baseConfig := func() *Config {
		return &Config{
			Receivers: map[component.ID]component.Config{
				component.MustNewID("otlp"): &opaqueConfig{Token: "secret1"},
			},
			Processors: map[component.ID]component.Config{},
			Exporters: map[component.ID]component.Config{
				component.MustNewID("otlp"): &opaqueConfig{Token: "secret1"},
			},
			Extensions: map[component.ID]component.Config{},
			Connectors: map[component.ID]component.Config{},
			Service: service.Config{
				Pipelines: pipelines.Config{
					pipeline.NewID(pipeline.SignalTraces): {
						Receivers: []component.ID{component.MustNewID("otlp")},
						Exporters: []component.ID{component.MustNewID("otlp")},
					},
				},
			},
		}
	}

	t.Run("receiver_secret_changed_is_receiver_only", func(t *testing.T) {
		newCfg := baseConfig()
		newCfg.Receivers[component.MustNewID("otlp")] = &opaqueConfig{Token: "secret2"}
		assert.True(t, receiversOnlyChange(baseConfig(), newCfg, func(component.ID) bool { return false }))
	})

	t.Run("exporter_secret_changed_is_not_receiver_only", func(t *testing.T) {
		newCfg := baseConfig()
		newCfg.Exporters[component.MustNewID("otlp")] = &opaqueConfig{Token: "secret2"}
		// Despite both secrets redacting to the same text, the diff must detect
		// the exporter change and fall back to a full reload.
		assert.False(t, receiversOnlyChange(baseConfig(), newCfg, func(component.ID) bool { return false }))
	})
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
