// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package migration

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	xotelconf "go.opentelemetry.io/contrib/otelconf/x"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestUnmarshalLogsConfigV120(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v1.2.0_logs.yaml"))
	require.NoError(t, err)

	cfg := LogsConfigV120{}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Processors, 3)
	// check the endpoint is prefixed w/ https
	require.Equal(t, "https://127.0.0.1:4317", *cfg.Processors[0].Batch.Exporter.OTLPHttp.Endpoint)
	// check the endpoint is prefixed w/ http
	require.Equal(t, "http://127.0.0.1:4317", *cfg.Processors[2].Simple.Exporter.OTLPHttp.Endpoint)
	// ensure migration flag is not set for native v0.3.0 config
	require.False(t, cfg.MigratedFromV02)
}

func TestMarshalLogsConfigV120(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v1.2.0_logs.yaml"))
	require.NoError(t, err)
	cm2, err := confmaptest.LoadConf(filepath.Join("testdata", "v1.2.0_logs_marshaled.yaml"))
	require.NoError(t, err)

	cfg := LogsConfigV120{}
	require.NoError(t, cm.Unmarshal(&cfg))
	cm3 := confmap.New()
	require.NoError(t, cm3.Marshal(&cfg))

	assert.Equal(t, cm2.ToStringMap(), cm3.ToStringMap())
}

func TestUnmarshalTracesConfigV120(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v1.2.0_traces.yaml"))
	require.NoError(t, err)

	cfg := TracesConfigV120{}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Processors, 3)
	// check the endpoint is prefixed w/ https
	require.Equal(t, "https://127.0.0.1:4317", *cfg.Processors[0].Batch.Exporter.OTLPHttp.Endpoint)
	// check the endpoint is prefixed w/ http
	require.Equal(t, "http://127.0.0.1:4317", *cfg.Processors[2].Simple.Exporter.OTLPHttp.Endpoint)
	// ensure migration flag is not set for native v0.3.0 config
	require.False(t, cfg.MigratedFromV02)
	require.False(t, cfg.MigratedFromV03)
}

func TestMarshalTracesConfigV120(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v1.2.0_traces.yaml"))
	require.NoError(t, err)
	cm2, err := confmaptest.LoadConf(filepath.Join("testdata", "v1.2.0_traces_marshaled.yaml"))
	require.NoError(t, err)

	cfg := TracesConfigV120{}
	require.NoError(t, cm.Unmarshal(&cfg))
	cm3 := confmap.New()
	require.NoError(t, cm3.Marshal(&cfg))

	assert.Equal(t, cm2.ToStringMap(), cm3.ToStringMap())
}

func TestUnmarshalMetricsConfigV120(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v1.2.0_metrics.yaml"))
	require.NoError(t, err)

	cfg := MetricsConfigV120{}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Readers, 3)

	// check the endpoint is prefixed w/ https
	require.Equal(t, "https://127.0.0.1:4317", *cfg.Readers[0].Periodic.Exporter.OTLPHttp.Endpoint)
	// ensure migration flag is not set for native v0.3.0 config
	require.False(t, cfg.MigratedFromV02)

	// check that Prometheus exporter defaults are applied even when not explicitly set
	prom := cfg.Readers[1].Pull.Exporter.PrometheusDevelopment
	require.NotNil(t, prom)
	require.NotNil(t, prom.WithoutScopeInfo, "WithoutScopeInfo should default to non-nil")
	require.True(t, *prom.WithoutScopeInfo, "WithoutScopeInfo should default to true")
	// TODO: add tests for additional fields
}

func TestUnmarshalMetricsConfigV120_PrometheusExplicitFalse(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v1.2.0_metrics_prom_explicit.yaml"))
	require.NoError(t, err)

	cfg := MetricsConfigV120{}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Readers, 1)

	// When explicitly set to false, values should remain false
	prom := cfg.Readers[0].Pull.Exporter.PrometheusDevelopment
	require.NotNil(t, prom)
	require.NotNil(t, prom.WithoutScopeInfo)
	require.False(t, *prom.WithoutScopeInfo, "WithoutScopeInfo should be false when explicitly set")
	// TODO: add tests for additional fields
}

func TestMarshalMetricsConfigV120(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v1.2.0_metrics.yaml"))
	require.NoError(t, err)
	cm2, err := confmaptest.LoadConf(filepath.Join("testdata", "v1.2.0_metrics_marshaled.yaml"))
	require.NoError(t, err)

	cfg := MetricsConfigV120{}
	require.NoError(t, cm.Unmarshal(&cfg))
	cm3 := confmap.New()
	require.NoError(t, cm3.Marshal(&cfg))

	assert.Equal(t, cm2.ToStringMap(), cm3.ToStringMap())
}

func TestResourceConfigV120UnmarshalLegacyFormat(t *testing.T) {
	t.Run("unknown keys treated as legacy", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"service.name": "my-service",
		})
		var cfg ResourceConfigV120
		require.NoError(t, conf.Unmarshal(&cfg))
		assert.Empty(t, cfg.Attributes)
		assert.Equal(t, "my-service", cfg.LegacyAttributes["service.name"])
	})

	t.Run("schema_url declared", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"schema_url": "https://opentelemetry.io/schemas/1.38.0",
		})
		var cfg ResourceConfigV120
		require.NoError(t, conf.Unmarshal(&cfg))
		assert.Empty(t, cfg.Attributes)
		require.NotNil(t, cfg.SchemaUrl)
		assert.Equal(t, "https://opentelemetry.io/schemas/1.38.0", *cfg.SchemaUrl)
	})

	t.Run("no declarative keys", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"custom.attr": "value",
		})
		var cfg ResourceConfigV120
		require.NoError(t, conf.Unmarshal(&cfg))
		assert.Empty(t, cfg.Attributes)
		assert.Equal(t, "value", cfg.LegacyAttributes["custom.attr"])
	})

	t.Run("legacy with removed values", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"service.name":  nil,
			"service.other": "value",
		})
		var cfg ResourceConfigV120
		require.NoError(t, conf.Unmarshal(&cfg))
		assert.Equal(t, "value", cfg.LegacyAttributes["service.other"])
	})

	t.Run("legacy validation error", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"bad.value": map[string]any{"nested": "value"},
		})
		var cfg ResourceConfigV120
		require.NoError(t, conf.Unmarshal(&cfg))
		require.Error(t, confmap.Validate(&cfg))
	})
}

func TestResourceConfigV120UnmarshalDeclarativeFormat(t *testing.T) {
	t.Run("attributes list", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"attributes": []any{
				map[string]any{"name": "service.name", "value": "svc"},
			},
		})
		var cfg ResourceConfigV120
		require.NoError(t, conf.Unmarshal(&cfg))
		assert.Len(t, cfg.Attributes, 1)
		assert.Equal(t, "service.name", cfg.Attributes[0].Name)
		assert.Equal(t, "svc", cfg.Attributes[0].Value)
		assert.Empty(t, cfg.LegacyAttributes)
	})

	t.Run("schema_url only", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"schema_url": "https://opentelemetry.io/schemas/1.38.0",
		})
		var cfg ResourceConfigV120
		require.NoError(t, conf.Unmarshal(&cfg))
		assert.Empty(t, cfg.Attributes)
		require.NotNil(t, cfg.SchemaUrl)
		assert.Equal(t, "https://opentelemetry.io/schemas/1.38.0", *cfg.SchemaUrl)
	})

	t.Run("schema keys with legacy attributes", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"schema_url":  "https://opentelemetry.io/schemas/1.38.0",
			"legacy.attr": "value",
			"remove.attr": nil,
		})
		var cfg ResourceConfigV120
		require.NoError(t, conf.Unmarshal(&cfg))
		assert.Equal(t, "value", cfg.LegacyAttributes["legacy.attr"])
	})

	t.Run("attributes with legacy attributes validation error", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"attributes":  []any{map[string]any{"name": "service.name", "value": "svc"}},
			"legacy.attr": "value",
		})
		var cfg ResourceConfigV120
		require.NoError(t, conf.Unmarshal(&cfg))
		require.ErrorContains(t, confmap.Validate(&cfg), "resource::attributes cannot be used together with legacy inline resource attributes")
	})

	t.Run("declarative only with forward-compatible detectors", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"detectors": map[string]any{
				"attributes": map[string]any{
					"included": []any{"host"},
				},
			},
		})
		var cfg ResourceConfigV120
		require.NoError(t, conf.Unmarshal(&cfg))
		assert.Empty(t, cfg.Attributes)
	})

	t.Run("experimental detection development", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"detection/development": map[string]any{
				"detectors": []any{
					map[string]any{"host": map[string]any{}},
				},
			},
		})
		var cfg ResourceConfigV120
		require.NoError(t, conf.Unmarshal(&cfg))
		require.NotNil(t, cfg.DetectionDevelopment)
		require.Len(t, cfg.DetectionDevelopment.Detectors, 1)
		assert.NotNil(t, cfg.DetectionDevelopment.Detectors[0].Host)
		require.NoError(t, confmap.Validate(&cfg))
	})

	t.Run("experimental detection development marshals", func(t *testing.T) {
		cfg := ResourceConfigV120{
			DetectionDevelopment: &xotelconf.ExperimentalResourceDetection{
				Detectors: []xotelconf.ExperimentalResourceDetector{
					{Host: xotelconf.ExperimentalHostResourceDetector{}},
				},
			},
			LegacyAttributes: map[string]any{
				"foo": "bar",
			},
		}

		cm := confmap.New()
		require.NoError(t, cm.Marshal(&cfg))

		raw := cm.ToStringMap()
		detectionRaw, ok := raw["detection/development"].(map[string]any)
		require.True(t, ok)
		assert.NotContains(t, detectionRaw, "attributes")
		assert.Equal(t, "bar", raw["foo"])
	})

	t.Run("nil raw map", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any(nil))
		var cfg ResourceConfigV120
		require.NoError(t, conf.Unmarshal(&cfg))
		assert.Empty(t, cfg.Attributes)
	})

	t.Run("declarative unmarshal error", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"attributes_list": 123,
		})
		var cfg ResourceConfigV120
		require.Error(t, conf.Unmarshal(&cfg))
	})

	t.Run("declarative null value allowed", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"attributes": []any{
				map[string]any{"name": "service.name", "value": nil},
			},
		})
		var cfg ResourceConfigV120
		require.NoError(t, conf.Unmarshal(&cfg))
		require.NoError(t, confmap.Validate(&cfg))
	})
}

func TestRewritePrometheusDevelopment(t *testing.T) {
	newMetrics := func(key string) map[string]any {
		return map[string]any{"readers": []any{
			map[string]any{"periodic": map[string]any{}},
			map[string]any{"pull": map[string]any{"exporter": map[string]any{key: map[string]any{"port": 1}}}},
		}}
	}

	t.Run("stable to development and back", func(t *testing.T) {
		m := newMetrics("prometheus")
		renamed, err := rewritePrometheusDevelopment(m)
		require.NoError(t, err)
		assert.Equal(t, map[int]bool{1: true}, renamed)
		assert.Equal(t, newMetrics("prometheus/development"), m)

		require.NoError(t, rewritePrometheusStable(m, renamed))
		assert.Equal(t, newMetrics("prometheus"), m)
	})

	t.Run("development key is left alone when stable was not used", func(t *testing.T) {
		m := newMetrics("prometheus/development")
		renamed, err := rewritePrometheusDevelopment(m)
		require.NoError(t, err)
		assert.Empty(t, renamed)

		require.NoError(t, rewritePrometheusStable(m, renamed))
		assert.Equal(t, newMetrics("prometheus/development"), m)
	})

	t.Run("both keys set", func(t *testing.T) {
		m := newMetrics("prometheus")
		exp := m["readers"].([]any)[1].(map[string]any)["pull"].(map[string]any)["exporter"].(map[string]any)
		exp["prometheus/development"] = map[string]any{}
		_, err := rewritePrometheusDevelopment(m)
		require.Error(t, err)
	})

	t.Run("no readers", func(t *testing.T) {
		renamed, err := rewritePrometheusDevelopment(map[string]any{})
		require.NoError(t, err)
		assert.Empty(t, renamed)
	})
}

func TestMetricsConfigV120_PrometheusKeyRoundTrip(t *testing.T) {
	for _, key := range []string{"prometheus", "prometheus/development"} {
		t.Run(key, func(t *testing.T) {
			conf := confmap.NewFromStringMap(map[string]any{
				"readers": []any{map[string]any{"pull": map[string]any{"exporter": map[string]any{
					key: map[string]any{"host": "127.0.0.1", "port": 8888},
				}}}},
			})
			var cfg MetricsConfigV120
			require.NoError(t, conf.Unmarshal(&cfg))
			require.Len(t, cfg.Readers, 1)
			require.NotNil(t, cfg.Readers[0].Pull.Exporter.PrometheusDevelopment)
			assert.Equal(t, 8888, *cfg.Readers[0].Pull.Exporter.PrometheusDevelopment.Port)
			assert.Equal(t, key == "prometheus", cfg.PrometheusStableKeyReaders[0])

			// Marshaling emits the key the user wrote.
			out := confmap.New()
			require.NoError(t, out.Marshal(&cfg))
			exporter := out.ToStringMap()["readers"].([]any)[0].(map[string]any)["pull"].(map[string]any)["exporter"].(map[string]any)
			assert.Contains(t, exporter, key)
			assert.Len(t, exporter, 1)
		})
	}
}
