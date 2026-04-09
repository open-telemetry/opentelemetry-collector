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
			"attributes":  []any{map[string]any{"name": "service.name", "value": "svc"}},
			"legacy.attr": "value",
			"remove.attr": nil,
		})
		var cfg ResourceConfigV030
		require.NoError(t, conf.Unmarshal(&cfg))
		assert.Len(t, cfg.Attributes, 1)
		assert.Equal(t, "value", cfg.LegacyAttributes["legacy.attr"])
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
}
