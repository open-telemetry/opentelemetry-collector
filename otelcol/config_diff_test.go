// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/pipelines"
)

// baseRawConf is the raw (pre-decode) configuration matching baseServiceConfig.
func baseRawConf() map[string]any {
	return map[string]any{
		"receivers":  map[string]any{"otlp": map[string]any{}},
		"processors": map[string]any{"batch": map[string]any{}},
		"exporters":  map[string]any{"otlp": map[string]any{}},
		"extensions": map[string]any{},
		"connectors": map[string]any{},
		"service":    map[string]any{"telemetry": map[string]any{}},
	}
}

// baseServiceConfig is the typed plumbing (pipelines, enabled extensions)
// matching baseRawConf. Fingerprinting reads pipeline/extension membership
// from here rather than from the raw conf.
func baseServiceConfig() *Config {
	return &Config{
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

func fingerprintFor(t *testing.T, raw map[string]any, cfg *Config) configFingerprint {
	t.Helper()
	fp, err := fingerprintForPartialReload(confmap.NewFromStringMap(raw), cfg)
	require.NoError(t, err)
	return fp
}

func TestReceiversOnlyChanged(t *testing.T) {
	connectorID := component.MustNewID("forward")
	neverConnector := func(component.ID) bool { return false }
	isForwardConnector := func(id component.ID) bool { return id == connectorID }

	tests := []struct {
		name        string
		oldRaw      map[string]any
		oldCfg      *Config
		newRaw      map[string]any
		newCfg      *Config
		isConnector func(component.ID) bool
		want        bool
	}{
		{
			name:        "identical_configs",
			oldRaw:      baseRawConf(),
			oldCfg:      baseServiceConfig(),
			newRaw:      baseRawConf(),
			newCfg:      baseServiceConfig(),
			isConnector: neverConnector,
			want:        true,
		},
		{
			name:   "receiver_config_changed",
			oldRaw: baseRawConf(),
			oldCfg: baseServiceConfig(),
			newRaw: func() map[string]any {
				r := baseRawConf()
				r["receivers"] = map[string]any{"otlp": map[string]any{"endpoint": "changed"}}
				return r
			}(),
			newCfg:      baseServiceConfig(),
			isConnector: neverConnector,
			want:        true,
		},
		{
			name:   "receiver_added_to_config_map",
			oldRaw: baseRawConf(),
			oldCfg: baseServiceConfig(),
			newRaw: func() map[string]any {
				r := baseRawConf()
				r["receivers"] = map[string]any{
					"otlp":   map[string]any{},
					"jaeger": map[string]any{},
				}
				return r
			}(),
			newCfg:      baseServiceConfig(),
			isConnector: neverConnector,
			want:        true,
		},
		{
			name:   "receiver_added_to_pipeline",
			oldRaw: baseRawConf(),
			oldCfg: baseServiceConfig(),
			newRaw: func() map[string]any {
				r := baseRawConf()
				r["receivers"] = map[string]any{
					"otlp":   map[string]any{},
					"jaeger": map[string]any{},
				}
				return r
			}(),
			newCfg: func() *Config {
				c := baseServiceConfig()
				pid := pipeline.NewID(pipeline.SignalTraces)
				c.Service.Pipelines[pid].Receivers = append(c.Service.Pipelines[pid].Receivers, component.MustNewID("jaeger"))
				return c
			}(),
			isConnector: neverConnector,
			want:        true,
		},
		{
			name:   "processor_config_changed",
			oldRaw: baseRawConf(),
			oldCfg: baseServiceConfig(),
			newRaw: func() map[string]any {
				r := baseRawConf()
				r["processors"] = map[string]any{"batch": map[string]any{"timeout": "5s"}}
				return r
			}(),
			newCfg:      baseServiceConfig(),
			isConnector: neverConnector,
			want:        false,
		},
		{
			name:   "exporter_config_changed",
			oldRaw: baseRawConf(),
			oldCfg: baseServiceConfig(),
			newRaw: func() map[string]any {
				r := baseRawConf()
				r["exporters"] = map[string]any{"otlp": map[string]any{"endpoint": "changed"}}
				return r
			}(),
			newCfg:      baseServiceConfig(),
			isConnector: neverConnector,
			want:        false,
		},
		{
			name:   "pipeline_added",
			oldRaw: baseRawConf(),
			oldCfg: baseServiceConfig(),
			newRaw: baseRawConf(),
			newCfg: func() *Config {
				c := baseServiceConfig()
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
			name:   "pipeline_removed",
			oldRaw: baseRawConf(),
			oldCfg: func() *Config {
				c := baseServiceConfig()
				c.Service.Pipelines[pipeline.NewID(pipeline.SignalMetrics)] = &pipelines.PipelineConfig{
					Receivers:  []component.ID{component.MustNewID("otlp")},
					Processors: []component.ID{component.MustNewID("batch")},
					Exporters:  []component.ID{component.MustNewID("otlp")},
				}
				return c
			}(),
			newRaw:      baseRawConf(),
			newCfg:      baseServiceConfig(),
			isConnector: neverConnector,
			want:        false,
		},
		{
			name: "pipeline_processors_list_changed",
			oldRaw: func() map[string]any {
				r := baseRawConf()
				r["processors"] = map[string]any{"batch": map[string]any{}, "batch/2": map[string]any{}}
				return r
			}(),
			oldCfg: baseServiceConfig(),
			newRaw: func() map[string]any {
				r := baseRawConf()
				r["processors"] = map[string]any{"batch": map[string]any{}, "batch/2": map[string]any{}}
				return r
			}(),
			newCfg: func() *Config {
				c := baseServiceConfig()
				pid := pipeline.NewID(pipeline.SignalTraces)
				c.Service.Pipelines[pid].Processors = []component.ID{
					component.MustNewID("batch"),
					component.MustNewIDWithName("batch", "2"),
				}
				return c
			}(),
			isConnector: neverConnector,
			want:        false,
		},
		{
			name: "pipeline_exporters_list_changed",
			oldRaw: func() map[string]any {
				r := baseRawConf()
				r["exporters"] = map[string]any{"otlp": map[string]any{}, "otlp/2": map[string]any{}}
				return r
			}(),
			oldCfg: baseServiceConfig(),
			newRaw: func() map[string]any {
				r := baseRawConf()
				r["exporters"] = map[string]any{"otlp": map[string]any{}, "otlp/2": map[string]any{}}
				return r
			}(),
			newCfg: func() *Config {
				c := baseServiceConfig()
				pid := pipeline.NewID(pipeline.SignalTraces)
				c.Service.Pipelines[pid].Exporters = []component.ID{
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
			oldRaw: func() map[string]any {
				r := baseRawConf()
				r["connectors"] = map[string]any{"forward": map[string]any{}}
				return r
			}(),
			oldCfg: baseServiceConfig(),
			newRaw: func() map[string]any {
				r := baseRawConf()
				r["connectors"] = map[string]any{"forward": map[string]any{"changed": true}}
				return r
			}(),
			newCfg:      baseServiceConfig(),
			isConnector: isForwardConnector,
			want:        false,
		},
		{
			name: "connector_as_receiver_changed",
			oldRaw: func() map[string]any {
				r := baseRawConf()
				r["connectors"] = map[string]any{"forward": map[string]any{}}
				return r
			}(),
			oldCfg: baseServiceConfig(),
			newRaw: func() map[string]any {
				r := baseRawConf()
				r["connectors"] = map[string]any{"forward": map[string]any{}}
				return r
			}(),
			newCfg: func() *Config {
				c := baseServiceConfig()
				pid := pipeline.NewID(pipeline.SignalTraces)
				c.Service.Pipelines[pid].Receivers = append(c.Service.Pipelines[pid].Receivers, connectorID)
				return c
			}(),
			isConnector: isForwardConnector,
			want:        false,
		},
		{
			name: "extension_config_changed",
			oldRaw: func() map[string]any {
				r := baseRawConf()
				r["extensions"] = map[string]any{"health": map[string]any{}}
				return r
			}(),
			oldCfg: baseServiceConfig(),
			newRaw: func() map[string]any {
				r := baseRawConf()
				r["extensions"] = map[string]any{"health": map[string]any{"changed": true}}
				return r
			}(),
			newCfg:      baseServiceConfig(),
			isConnector: neverConnector,
			want:        false,
		},
		{
			name:   "extensions_list_changed",
			oldRaw: baseRawConf(),
			oldCfg: baseServiceConfig(),
			newRaw: func() map[string]any {
				r := baseRawConf()
				r["extensions"] = map[string]any{"health": map[string]any{}}
				return r
			}(),
			newCfg: func() *Config {
				c := baseServiceConfig()
				c.Service.Extensions = []component.ID{component.MustNewID("health")}
				return c
			}(),
			isConnector: neverConnector,
			want:        false,
		},
		{
			name: "receiver_removed_from_config_map",
			oldRaw: func() map[string]any {
				r := baseRawConf()
				r["receivers"] = map[string]any{"otlp": map[string]any{}, "jaeger": map[string]any{}}
				return r
			}(),
			oldCfg:      baseServiceConfig(),
			newRaw:      baseRawConf(),
			newCfg:      baseServiceConfig(),
			isConnector: neverConnector,
			want:        true,
		},
		{
			name:   "telemetry_changed",
			oldRaw: baseRawConf(),
			oldCfg: baseServiceConfig(),
			newRaw: func() map[string]any {
				r := baseRawConf()
				r["service"] = map[string]any{"telemetry": map[string]any{"logs": map[string]any{"level": "debug"}}}
				return r
			}(),
			newCfg:      baseServiceConfig(),
			isConnector: neverConnector,
			want:        false,
		},
		{
			name:   "pipeline_id_replaced",
			oldRaw: baseRawConf(),
			oldCfg: func() *Config {
				c := baseServiceConfig()
				c.Service.Pipelines[pipeline.NewID(pipeline.SignalMetrics)] = &pipelines.PipelineConfig{
					Receivers:  []component.ID{component.MustNewID("otlp")},
					Processors: []component.ID{component.MustNewID("batch")},
					Exporters:  []component.ID{component.MustNewID("otlp")},
				}
				return c
			}(),
			newRaw: baseRawConf(),
			newCfg: func() *Config {
				c := baseServiceConfig()
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
			old := fingerprintFor(t, tt.oldRaw, tt.oldCfg)
			cur := fingerprintFor(t, tt.newRaw, tt.newCfg)
			got := receiversOnlyChanged(old, cur, tt.isConnector)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestReceiversOnlyChangedOpaqueSecrets verifies that the diff observes
// changes to secret values. Fingerprints hash the raw, pre-decode configuration
// map, real secret string values, not a configopaque.String-wrapped and redacted
// form.
func TestReceiversOnlyChangedOpaqueSecrets(t *testing.T) {
	baseRaw := func(secret string) map[string]any {
		r := baseRawConf()
		r["receivers"] = map[string]any{"otlp": map[string]any{"token": secret}}
		return r
	}
	neverConnector := func(component.ID) bool { return false }

	t.Run("receiver_secret_changed_is_receiver_only", func(t *testing.T) {
		old := fingerprintFor(t, baseRaw("secret1"), baseServiceConfig())
		cur := fingerprintFor(t, baseRaw("secret2"), baseServiceConfig())
		assert.True(t, receiversOnlyChanged(old, cur, neverConnector))
	})

	t.Run("exporter_secret_changed_is_not_receiver_only", func(t *testing.T) {
		oldRaw := baseRawConf()
		oldRaw["exporters"] = map[string]any{"otlp": map[string]any{"token": "secret1"}}
		newRaw := baseRawConf()
		newRaw["exporters"] = map[string]any{"otlp": map[string]any{"token": "secret2"}}

		old := fingerprintFor(t, oldRaw, baseServiceConfig())
		cur := fingerprintFor(t, newRaw, baseServiceConfig())
		assert.False(t, receiversOnlyChanged(old, cur, neverConnector))
	})
}

// TestFingerprintForPartialReload_IgnoresDecodeLossiness verifies where a
// receiver's config fields tagged mapstructure:"-" works.
func TestFingerprintForPartialReload_IgnoresDecodeLossiness(t *testing.T) {
	// scrapers is analogous to a mapstructure:"-" field: content that a real
	// decode might not preserve via generic mapstructure but that is present
	// verbatim in the raw source map either way.
	rawWithScrapers := func() map[string]any {
		r := baseRawConf()
		r["receivers"] = map[string]any{
			"hostmetrics": map[string]any{"scrapers": map[string]any{"cpu": map[string]any{}}},
		}
		return r
	}

	old := fingerprintFor(t, rawWithScrapers(), baseServiceConfig())
	cur := fingerprintFor(t, rawWithScrapers(), baseServiceConfig())
	assert.Equal(t, old.receiverHashes, cur.receiverHashes,
		"identical raw content, including a mapstructure:\"-\"-style field, must fingerprint identically")

	changedRaw := rawWithScrapers()
	changedRaw["receivers"] = map[string]any{
		"hostmetrics": map[string]any{"scrapers": map[string]any{"cpu": map[string]any{}, "disk": map[string]any{}}},
	}
	changed := fingerprintFor(t, changedRaw, baseServiceConfig())
	assert.NotEqual(t, old.receiverHashes, changed.receiverHashes,
		"a real change to the mapstructure:\"-\"-style field must still be observed")
}

// TestFingerprintForPartialReload_ImmuneToLiveMutation verifies that
// mutating a decoded component.Config value after fingerprinting (modeling a
// running component mutating the config it was built from, e.g. resolving a
// default or normalizing a value) cannot affect a fingerprint already taken.
func TestFingerprintForPartialReload_ImmuneToLiveMutation(t *testing.T) {
	type mutableConfig struct {
		Endpoint string
	}

	raw := baseRawConf()
	cfg := baseServiceConfig()
	cfg.Receivers = map[component.ID]component.Config{
		component.MustNewID("otlp"): &mutableConfig{Endpoint: "original"},
	}

	before := fingerprintFor(t, raw, cfg)

	// Simulate a running component mutating the config object it was built
	// from, in place, after Start.
	cfg.Receivers[component.MustNewID("otlp")].(*mutableConfig).Endpoint = "mutated-by-live-component"

	after, err := fingerprintForPartialReload(confmap.NewFromStringMap(raw), cfg)
	require.NoError(t, err)
	assert.Equal(t, before, after, "mutating a live component.Config after fingerprinting must not change the fingerprint")
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

func TestChangedReceivers(t *testing.T) {
	old := fingerprintFor(t, baseRawConf(), baseServiceConfig())

	newRaw := baseRawConf()
	newRaw["receivers"] = map[string]any{
		"otlp":   map[string]any{"endpoint": "changed"}, // present in both, changed
		"jaeger": map[string]any{},                      // added: not in old, must not appear as "changed"
	}
	cur := fingerprintFor(t, newRaw, baseServiceConfig())

	changed := changedReceivers(old, cur)
	assert.Equal(t, map[component.ID]bool{component.MustNewID("otlp"): true}, changed)
}

func TestProcessorsOnlyChanged(t *testing.T) {
	withRaw := func(mutate func(map[string]any)) map[string]any {
		r := baseRawConf()
		mutate(r)
		return r
	}
	withCfg := func(mutate func(*Config)) *Config {
		c := baseServiceConfig()
		mutate(c)
		return c
	}
	tracesID := pipeline.NewID(pipeline.SignalTraces)

	tests := []struct {
		name   string
		newRaw map[string]any
		newCfg *Config
		want   bool
	}{
		{"identical", baseRawConf(), baseServiceConfig(), true},
		{
			"processor_config_changed",
			withRaw(func(r map[string]any) { r["processors"] = map[string]any{"batch": map[string]any{"timeout": "1s"}} }),
			baseServiceConfig(), true,
		},
		{
			"processor_added_to_config_map",
			withRaw(func(r map[string]any) {
				r["processors"] = map[string]any{"batch": map[string]any{}, "memory_limiter": map[string]any{}}
			}),
			baseServiceConfig(), true,
		},
		{
			"processor_added_to_pipeline",
			baseRawConf(),
			withCfg(func(c *Config) {
				p := c.Service.Pipelines[tracesID]
				p.Processors = append(p.Processors, component.MustNewID("memory_limiter"))
			}),
			true,
		},
		{
			"processor_removed_from_pipeline",
			baseRawConf(),
			withCfg(func(c *Config) { c.Service.Pipelines[tracesID].Processors = nil }),
			true,
		},
		{
			"receiver_config_changed",
			withRaw(func(r map[string]any) { r["receivers"] = map[string]any{"otlp": map[string]any{"endpoint": "x"}} }),
			baseServiceConfig(), false,
		},
		{
			"receiver_added_to_pipeline",
			baseRawConf(),
			withCfg(func(c *Config) {
				p := c.Service.Pipelines[tracesID]
				p.Receivers = append(p.Receivers, component.MustNewID("jaeger"))
			}),
			false,
		},
		{
			"exporter_config_changed",
			withRaw(func(r map[string]any) { r["exporters"] = map[string]any{"otlp": map[string]any{"endpoint": "x"}} }),
			baseServiceConfig(), false,
		},
		{
			"exporter_added_to_pipeline",
			baseRawConf(),
			withCfg(func(c *Config) {
				p := c.Service.Pipelines[tracesID]
				p.Exporters = append(p.Exporters, component.MustNewID("debug"))
			}),
			false,
		},
		{
			"connector_config_changed",
			withRaw(func(r map[string]any) { r["connectors"] = map[string]any{"forward": map[string]any{}} }),
			baseServiceConfig(), false,
		},
		{
			"extension_config_changed",
			withRaw(func(r map[string]any) { r["extensions"] = map[string]any{"zpages": map[string]any{}} }),
			baseServiceConfig(), false,
		},
		{
			"extension_list_changed",
			baseRawConf(),
			withCfg(func(c *Config) { c.Service.Extensions = []component.ID{component.MustNewID("zpages")} }),
			false,
		},
		{
			"telemetry_changed",
			withRaw(func(r map[string]any) {
				r["service"] = map[string]any{"telemetry": map[string]any{"logs": map[string]any{"level": "debug"}}}
			}),
			baseServiceConfig(), false,
		},
		{
			"pipeline_added",
			baseRawConf(),
			withCfg(func(c *Config) {
				c.Service.Pipelines[pipeline.NewID(pipeline.SignalLogs)] = &pipelines.PipelineConfig{
					Receivers: []component.ID{component.MustNewID("otlp")},
					Exporters: []component.ID{component.MustNewID("otlp")},
				}
			}),
			false,
		},
		{
			"pipeline_removed",
			baseRawConf(),
			withCfg(func(c *Config) { c.Service.Pipelines = pipelines.Config{} }),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldFP := fingerprintFor(t, baseRawConf(), baseServiceConfig())
			newFP := fingerprintFor(t, tt.newRaw, tt.newCfg)
			assert.Equal(t, tt.want, processorsOnlyChanged(oldFP, newFP))
		})
	}
}

func TestReceiversOnlyChangedRejectsProcessorChange(t *testing.T) {
	newRaw := baseRawConf()
	newRaw["processors"] = map[string]any{"batch": map[string]any{"timeout": "1s"}}
	oldFP := fingerprintFor(t, baseRawConf(), baseServiceConfig())
	newFP := fingerprintFor(t, newRaw, baseServiceConfig())
	assert.False(t, receiversOnlyChanged(oldFP, newFP, func(component.ID) bool { return false }))
}

func TestChangedProcessors(t *testing.T) {
	oldRaw := baseRawConf()
	oldRaw["processors"] = map[string]any{"a": map[string]any{}, "b": map[string]any{}, "gone": map[string]any{}}
	newRaw := baseRawConf()
	newRaw["processors"] = map[string]any{"a": map[string]any{}, "b": map[string]any{"x": 1}, "new": map[string]any{}}

	got := changedProcessors(fingerprintFor(t, oldRaw, baseServiceConfig()), fingerprintFor(t, newRaw, baseServiceConfig()))
	assert.Equal(t, map[component.ID]bool{component.MustNewID("b"): true}, got)
}

func TestPartialReloadSupported(t *testing.T) {
	never := func(component.ID) bool { return false }
	tracesID := pipeline.NewID(pipeline.SignalTraces)

	// Both a receiver and a processor change.
	bothRaw := baseRawConf()
	bothRaw["receivers"] = map[string]any{"otlp": map[string]any{"endpoint": "x"}}
	bothRaw["processors"] = map[string]any{"batch": map[string]any{"timeout": "1s"}}
	bothCfg := baseServiceConfig()
	bothCfg.Service.Pipelines[tracesID].Receivers = append(bothCfg.Service.Pipelines[tracesID].Receivers, component.MustNewID("jaeger"))
	bothCfg.Service.Pipelines[tracesID].Processors = nil

	// A receiver and an exporter change.
	exporterRaw := baseRawConf()
	exporterRaw["receivers"] = map[string]any{"otlp": map[string]any{"endpoint": "x"}}
	exporterRaw["exporters"] = map[string]any{"otlp": map[string]any{"endpoint": "x"}}

	oldFP := fingerprintFor(t, baseRawConf(), baseServiceConfig())
	bothFP := fingerprintFor(t, bothRaw, bothCfg)
	exporterFP := fingerprintFor(t, exporterRaw, baseServiceConfig())

	tests := []struct {
		name                            string
		cur                             configFingerprint
		allowReceivers, allowProcessors bool
		want                            bool
	}{
		{"both_changed_both_allowed", bothFP, true, true, true},
		{"both_changed_only_receivers_allowed", bothFP, true, false, false},
		{"both_changed_only_processors_allowed", bothFP, false, true, false},
		{"both_changed_none_allowed", bothFP, false, false, false},
		{"unchanged_none_allowed", oldFP, false, false, true},
		{"exporter_change_never_allowed", exporterFP, true, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, partialReloadSupported(oldFP, tt.cur, never, tt.allowReceivers, tt.allowProcessors))
		})
	}
}

func TestFingerprintForPartialReload_Errors(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(map[string]any)
		wantErr string
	}{
		{
			name:    "unencodable_other_section",
			mutate:  func(r map[string]any) { r["exporters"] = map[string]any{"otlp": make(chan int)} },
			wantErr: "could not fingerprint configuration",
		},
		{
			name:    "receivers_not_a_map",
			mutate:  func(r map[string]any) { r["receivers"] = "invalid" },
			wantErr: "could not fingerprint receiver configuration",
		},
		{
			name:    "processors_not_a_map",
			mutate:  func(r map[string]any) { r["processors"] = []any{"batch"} },
			wantErr: "could not fingerprint processor configuration",
		},
		{
			name:    "invalid_component_id",
			mutate:  func(r map[string]any) { r["processors"] = map[string]any{"bad id/": map[string]any{}} },
			wantErr: "invalid component id",
		},
		{
			name:    "unencodable_entry",
			mutate:  func(r map[string]any) { r["receivers"] = map[string]any{"otlp": make(chan int)} },
			wantErr: "could not encode value for fingerprint",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := baseRawConf()
			tt.mutate(raw)
			_, err := fingerprintForPartialReload(confmap.NewFromStringMap(raw), baseServiceConfig())
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
