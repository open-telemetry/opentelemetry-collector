// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package migration

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

func TestUnmarshalLogsConfigV030(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v0.3.0_logs.yaml"))
	require.NoError(t, err)

	cfg := LogsConfigV030{}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Processors, 3)
	// check the endpoint is prefixed w/ https
	require.Equal(t, "https://127.0.0.1:4317", *cfg.Processors[0].Batch.Exporter.OTLP.Endpoint)
	// check the endpoint is prefixed w/ http
	require.Equal(t, "http://127.0.0.1:4317", *cfg.Processors[2].Simple.Exporter.OTLP.Endpoint)
	// ensure migration flag is not set for native v0.3.0 config
	require.False(t, cfg.MigratedFromV02)
}

func TestMarshalLogsConfigV030(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v0.3.0_logs.yaml"))
	require.NoError(t, err)
	cm2, err := confmaptest.LoadConf(filepath.Join("testdata", "v0.3.0_logs_marshaled.yaml"))
	require.NoError(t, err)

	cfg := LogsConfigV030{}
	require.NoError(t, cm.Unmarshal(&cfg))
	cm3 := confmap.New()
	require.NoError(t, cm3.Marshal(&cfg))

	assert.Equal(t, cm2.ToStringMap(), cm3.ToStringMap())
}

func TestUnmarshalTracesConfigV030(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v0.3.0_traces.yaml"))
	require.NoError(t, err)

	cfg := TracesConfigV030{}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Processors, 3)
	// check the endpoint is prefixed w/ https
	require.Equal(t, "https://127.0.0.1:4317", *cfg.Processors[0].Batch.Exporter.OTLP.Endpoint)
	// check the endpoint is prefixed w/ http
	require.Equal(t, "http://127.0.0.1:4317", *cfg.Processors[2].Simple.Exporter.OTLP.Endpoint)
	// ensure migration flag is not set for native v0.3.0 config
	require.False(t, cfg.MigratedFromV02)
}

func TestMarshalTracesConfigV030(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v0.3.0_traces.yaml"))
	require.NoError(t, err)
	cm2, err := confmaptest.LoadConf(filepath.Join("testdata", "v0.3.0_traces_marshaled.yaml"))
	require.NoError(t, err)

	cfg := TracesConfigV030{}
	require.NoError(t, cm.Unmarshal(&cfg))
	cm3 := confmap.New()
	require.NoError(t, cm3.Marshal(&cfg))

	assert.Equal(t, cm2.ToStringMap(), cm3.ToStringMap())
}

func TestUnmarshalMetricsConfigV030(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v0.3.0_metrics.yaml"))
	require.NoError(t, err)

	cfg := MetricsConfigV030{}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Readers, 2)

	// check the endpoint is prefixed w/ https
	require.Equal(t, "https://127.0.0.1:4317", *cfg.Readers[0].Periodic.Exporter.OTLP.Endpoint)
	// ensure migration flag is not set for native v0.3.0 config
	require.False(t, cfg.MigratedFromV02)

	// check that Prometheus exporter defaults are applied even when not explicitly set
	prom := cfg.Readers[1].Pull.Exporter.Prometheus
	require.NotNil(t, prom)
	require.NotNil(t, prom.WithoutScopeInfo, "WithoutScopeInfo should default to non-nil")
	require.True(t, *prom.WithoutScopeInfo, "WithoutScopeInfo should default to true")
	require.NotNil(t, prom.WithoutUnits, "WithoutUnits should default to non-nil")
	require.True(t, *prom.WithoutUnits, "WithoutUnits should default to true")
	require.NotNil(t, prom.WithoutTypeSuffix, "WithoutTypeSuffix should default to non-nil")
	require.True(t, *prom.WithoutTypeSuffix, "WithoutTypeSuffix should default to true")
}

func TestUnmarshalMetricsConfigV030_PrometheusExplicitFalse(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v0.3.0_metrics_prom_explicit.yaml"))
	require.NoError(t, err)

	cfg := MetricsConfigV030{}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Readers, 1)

	// When explicitly set to false, values should remain false
	prom := cfg.Readers[0].Pull.Exporter.Prometheus
	require.NotNil(t, prom)
	require.NotNil(t, prom.WithoutScopeInfo)
	require.False(t, *prom.WithoutScopeInfo, "WithoutScopeInfo should be false when explicitly set")
	require.NotNil(t, prom.WithoutUnits)
	require.False(t, *prom.WithoutUnits, "WithoutUnits should be false when explicitly set")
	require.NotNil(t, prom.WithoutTypeSuffix)
	require.False(t, *prom.WithoutTypeSuffix, "WithoutTypeSuffix should be false when explicitly set")
}

func TestMarshalMetricsConfigV030(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v0.3.0_metrics.yaml"))
	require.NoError(t, err)
	cm2, err := confmaptest.LoadConf(filepath.Join("testdata", "v0.3.0_metrics_marshaled.yaml"))
	require.NoError(t, err)

	cfg := MetricsConfigV030{}
	require.NoError(t, cm.Unmarshal(&cfg))
	cm3 := confmap.New()
	require.NoError(t, cm3.Marshal(&cfg))

	assert.Equal(t, cm2.ToStringMap(), cm3.ToStringMap())
}

func TestResourceConfigV030UnmarshalLegacyFormat(t *testing.T) {
	t.Run("unknown keys treated as legacy", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"service.name": "my-service",
		})
		var cfg ResourceConfigV030
		require.NoError(t, conf.Unmarshal(&cfg))
		assert.Empty(t, cfg.Attributes)
		assert.Equal(t, "my-service", cfg.LegacyAttributes["service.name"])
	})

	t.Run("schema_url declared", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"schema_url": "https://opentelemetry.io/schemas/1.38.0",
		})
		var cfg ResourceConfigV030
		require.NoError(t, conf.Unmarshal(&cfg))
		assert.Empty(t, cfg.Attributes)
		require.NotNil(t, cfg.SchemaUrl)
		assert.Equal(t, "https://opentelemetry.io/schemas/1.38.0", *cfg.SchemaUrl)
	})

	t.Run("no declarative keys", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"custom.attr": "value",
		})
		var cfg ResourceConfigV030
		require.NoError(t, conf.Unmarshal(&cfg))
		assert.Empty(t, cfg.Attributes)
		assert.Equal(t, "value", cfg.LegacyAttributes["custom.attr"])
	})

	t.Run("legacy with removed values", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"service.name":  nil,
			"service.other": "value",
		})
		var cfg ResourceConfigV030
		require.NoError(t, conf.Unmarshal(&cfg))
		assert.Equal(t, "value", cfg.LegacyAttributes["service.other"])
	})

	t.Run("legacy validation error", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"bad.value": map[string]any{"nested": "value"},
		})
		var cfg ResourceConfigV030
		require.NoError(t, conf.Unmarshal(&cfg))
		require.Error(t, xconfmap.Validate(&cfg))
	})
}

func TestResourceConfigV030UnmarshalDeclarativeFormat(t *testing.T) {
	t.Run("attributes list", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"attributes": []any{
				map[string]any{"name": "service.name", "value": "svc"},
			},
		})
		var cfg ResourceConfigV030
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
		var cfg ResourceConfigV030
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
		var cfg ResourceConfigV030
		require.NoError(t, conf.Unmarshal(&cfg))
		assert.Equal(t, "value", cfg.LegacyAttributes["legacy.attr"])
	})

	t.Run("attributes with legacy attributes validation error", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"attributes":  []any{map[string]any{"name": "service.name", "value": "svc"}},
			"legacy.attr": "value",
		})
		var cfg ResourceConfigV030
		require.NoError(t, conf.Unmarshal(&cfg))
		require.ErrorContains(t, xconfmap.Validate(&cfg), "resource::attributes cannot be used together with legacy inline resource attributes")
	})

	t.Run("declarative only with detectors", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"detectors": map[string]any{
				"attributes": map[string]any{
					"included": []any{"host"},
				},
			},
		})
		var cfg ResourceConfigV030
		require.NoError(t, conf.Unmarshal(&cfg))
		assert.Empty(t, cfg.Attributes)
	})

	t.Run("nil raw map", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any(nil))
		var cfg ResourceConfigV030
		require.NoError(t, conf.Unmarshal(&cfg))
		assert.Empty(t, cfg.Attributes)
	})

	t.Run("declarative unmarshal error", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"attributes_list": 123,
		})
		var cfg ResourceConfigV030
		require.Error(t, conf.Unmarshal(&cfg))
	})

	t.Run("declarative null value allowed", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"attributes": []any{
				map[string]any{"name": "service.name", "value": nil},
			},
		})
		var cfg ResourceConfigV030
		require.NoError(t, conf.Unmarshal(&cfg))
		require.NoError(t, xconfmap.Validate(&cfg))
	})
}
