// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package migration

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
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

func TestUnmarshalMetricsConfigV030(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v0.3.0_metrics.yaml"))
	require.NoError(t, err)

	cfg := MetricsConfigV030{}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Readers, 2)

	// check the endpoint is prefixed w/ https
	require.Equal(t, "https://127.0.0.1:4317", *cfg.Readers[0].Periodic.Exporter.OTLP.Endpoint)
}

func TestResourceConfigV030UnmarshalLegacyFormat(t *testing.T) {
	t.Run("unknown keys treated as legacy", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"service.name": "my-service",
		})
		var cfg ResourceConfigV030
		require.NoError(t, cfg.Unmarshal(conf))
		assert.Len(t, cfg.Attributes, 1)
		assert.Equal(t, "service.name", cfg.Attributes[0].Name)
		assert.Equal(t, "my-service", cfg.Attributes[0].Value)
	})

	t.Run("reserved key with primitive treated as legacy", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"schema_url": "legacy-value",
		})
		var cfg ResourceConfigV030
		require.NoError(t, cfg.Unmarshal(conf))
		assert.Len(t, cfg.Attributes, 1)
		assert.Equal(t, "schema_url", cfg.Attributes[0].Name)
		assert.Equal(t, "legacy-value", cfg.Attributes[0].Value)
	})

	t.Run("no declarative keys", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"custom.attr": "value",
		})
		var cfg ResourceConfigV030
		require.NoError(t, cfg.Unmarshal(conf))
		assert.Len(t, cfg.Attributes, 1)
		assert.Equal(t, "custom.attr", cfg.Attributes[0].Name)
		assert.Equal(t, "value", cfg.Attributes[0].Value)
	})

	t.Run("legacy with removed values", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"service.name":  nil,
			"service.other": "value",
		})
		var cfg ResourceConfigV030
		require.NoError(t, cfg.Unmarshal(conf))
		assert.Len(t, cfg.Attributes, 1)
		assert.True(t, cfg.IsRemoved("service.name"))
	})

	t.Run("legacy unmarshal error", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"bad.value": map[string]any{"nested": "value"},
		})
		var cfg ResourceConfigV030
		require.Error(t, cfg.Unmarshal(conf))
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
		require.NoError(t, cfg.Unmarshal(conf))
		assert.Len(t, cfg.Attributes, 1)
		assert.Equal(t, "service.name", cfg.Attributes[0].Name)
		assert.Equal(t, "svc", cfg.Attributes[0].Value)
	})

	t.Run("schema_url only", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"schema_url": "https://opentelemetry.io/schemas/1.38.0",
		})
		var cfg ResourceConfigV030
		require.NoError(t, cfg.Unmarshal(conf))
		assert.Len(t, cfg.Attributes, 1)
		assert.Equal(t, "schema_url", cfg.Attributes[0].Name)
		assert.Equal(t, "https://opentelemetry.io/schemas/1.38.0", cfg.Attributes[0].Value)
	})

	t.Run("schema keys with legacy attributes", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"schema_url":  "https://opentelemetry.io/schemas/1.38.0",
			"attributes":  []any{map[string]any{"name": "service.name", "value": "svc"}},
			"legacy.attr": "value",
			"remove.attr": nil,
		})
		var cfg ResourceConfigV030
		require.NoError(t, cfg.Unmarshal(conf))
		assert.Len(t, cfg.Attributes, 3)
		assert.True(t, cfg.IsRemoved("remove.attr"))
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
		require.NoError(t, cfg.Unmarshal(conf))
		assert.Empty(t, cfg.Attributes)
		assert.False(t, cfg.IsRemoved("service.name"))
	})

	t.Run("nil conf", func(t *testing.T) {
		var cfg ResourceConfigV030
		require.NoError(t, cfg.Unmarshal(nil))
		assert.Empty(t, cfg.Attributes)
	})
	t.Run("nil raw map", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any(nil))
		var cfg ResourceConfigV030
		require.NoError(t, cfg.Unmarshal(conf))
		assert.Empty(t, cfg.Attributes)
	})

	t.Run("declarative unmarshal error", func(t *testing.T) {
		conf := confmap.NewFromStringMap(map[string]any{
			"attributes_list": 123,
		})
		var cfg ResourceConfigV030
		require.Error(t, cfg.Unmarshal(conf))
	})
}

func TestDeclarativeHelpers(t *testing.T) {
	t.Run("declarative config from raw", func(t *testing.T) {
		raw := map[string]any{
			"schema_url":  "https://opentelemetry.io/schemas/1.38.0",
			"attributes":  []any{map[string]any{"name": "service.name", "value": "svc"}},
			"legacy.attr": "value",
		}
		conf := declarativeConfigFromRaw(raw)
		var cfg config.Resource
		require.NoError(t, conf.Unmarshal(&cfg))
		assert.Len(t, cfg.Attributes, 1)
	})
	t.Run("declarative config from nil", func(t *testing.T) {
		conf := declarativeConfigFromRaw(nil)
		var cfg config.Resource
		require.NoError(t, conf.Unmarshal(&cfg))
		assert.Empty(t, cfg.Attributes)
	})

	t.Run("isDeclarativeResourceConfig cases", func(t *testing.T) {
		assert.False(t, isDeclarativeResourceConfig(nil))
		assert.False(t, isDeclarativeResourceConfig(map[string]any{}))
		assert.False(t, isDeclarativeResourceConfig(map[string]any{"schema_url": "legacy"}))
		assert.True(t, isDeclarativeResourceConfig(map[string]any{
			"schema_url": "https://opentelemetry.io/schemas/1.38.0",
			"attributes": []any{map[string]any{"name": "service.name", "value": "svc"}},
		}))
		assert.True(t, isDeclarativeResourceConfig(map[string]any{
			"attributes_list": "service.name=svc",
		}))
	})

	t.Run("isPrimitiveOnlyConfig", func(t *testing.T) {
		assert.True(t, isPrimitiveOnlyConfig(map[string]any{"schema_url": "value"}))
		assert.False(t, isPrimitiveOnlyConfig(map[string]any{"schema_url": map[string]any{}}))
		assert.False(t, isPrimitiveOnlyConfig(map[string]any{"schema_url": "value", "other": "v"}))
		assert.False(t, isPrimitiveOnlyConfig(map[string]any{}))
	})

	t.Run("isPrimitive", func(t *testing.T) {
		assert.True(t, isPrimitive("value"))
		assert.True(t, isPrimitive(true))
		assert.True(t, isPrimitive(1))
		assert.True(t, isPrimitive(1.2))
		assert.False(t, isPrimitive(map[string]any{}))
	})

	t.Run("legacyAttributesFromRaw", func(t *testing.T) {
		attrs, removed := legacyAttributesFromRaw(nil)
		assert.Nil(t, attrs)
		assert.Nil(t, removed)

		attrs, removed = legacyAttributesFromRaw(map[string]any{
			"schema_url":  "https://opentelemetry.io/schemas/1.38.0",
			"legacy.attr": "value",
			"remove.attr": nil,
		})
		assert.Len(t, attrs, 2)
		assert.Contains(t, removed, "remove.attr")
	})
}
