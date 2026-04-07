// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/service/telemetry"
	"go.opentelemetry.io/collector/service/telemetry/otelconftelemetry/internal/migration"
)

func TestCreateResource(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		var f otelconfFactory
		res, err := f.createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.Contains(t, raw, "service.instance.id")
		delete(raw, "service.instance.id") // remove since it's random
		assert.Equal(t, map[string]any{
			"service.name":    "otelcol",
			"service.version": "latest",
		}, raw)
	})
	t.Run("with resource attributes (legacy format)", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		legacy := confmap.NewFromStringMap(map[string]any{
			"extra.attr":          "value",
			"service.name":        "custom-service",
			"service.version":     "0.1.0",
			"service.instance.id": nil,
		})
		require.NoError(t, legacy.Unmarshal(&cfg.Resource))
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		var f otelconfFactory
		res, err := f.createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.Equal(t, map[string]any{
			"extra.attr":      "value",
			"service.name":    "custom-service",
			"service.version": "0.1.0",
		}, raw)
	})
	t.Run("with legacy format removing default attributes", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		legacy := confmap.NewFromStringMap(map[string]any{
			"custom.attr":         "value",
			"service.name":        nil,
			"service.version":     nil,
			"service.instance.id": nil,
		})
		require.NoError(t, legacy.Unmarshal(&cfg.Resource))
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		var f otelconfFactory
		res, err := f.createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.Equal(t, map[string]any{
			"custom.attr": "value",
		}, raw)
	})
	t.Run("with resource attributes (new format)", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		cfg.Resource.Attributes = []config.AttributeNameValue{
			{Name: "extra.attr", Value: "value"},
			{Name: "service.name", Value: "custom-service"},
			{Name: "service.version", Value: "0.1.0"},
			{Name: "service.instance.id", Value: nil},
		}
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		var f otelconfFactory
		res, err := f.createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.Equal(t, map[string]any{
			"extra.attr":          "value",
			"service.name":        "custom-service",
			"service.version":     "0.1.0",
			"service.instance.id": "<nil>",
		}, raw)
	})
	t.Run("with custom schema_url", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		customSchemaURL := "https://opentelemetry.io/schemas/1.39.0"
		cfg.Resource.SchemaUrl = &customSchemaURL
		cfg.Resource.Attributes = []config.AttributeNameValue{
			{Name: "service.name", Value: "test-service"},
		}
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		var f otelconfFactory
		res, err := f.createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.Contains(t, raw, "service.name")
		assert.Equal(t, "test-service", raw["service.name"])
		assert.Contains(t, raw, "service.instance.id")
		assert.Contains(t, raw, "service.version")
	})
	t.Run("with detectors for forward compatibility", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		cfg.Resource.Detectors = &config.Detectors{
			Attributes: &config.DetectorsAttributes{
				Included: []string{"host", "os"},
			},
		}
		cfg.Resource.Attributes = []config.AttributeNameValue{
			{Name: "service.name", Value: "test-service"},
		}
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		var f otelconfFactory
		res, err := f.createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.Contains(t, raw, "service.name")
		assert.Equal(t, "test-service", raw["service.name"])
	})
	t.Run("with typed attributes", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		cfg.Resource.Attributes = []config.AttributeNameValue{
			{Name: "bool.attr", Value: true},
			{Name: "int64.attr", Value: int64(42)},
			{Name: "int32.attr", Value: int32(-32)},
			{Name: "int16.attr", Value: int16(-16)},
			{Name: "int8.attr", Value: int8(-8)},
			{Name: "uint.attr", Value: uint(200)},
			{Name: "int.attr", Value: int(123)},
			{Name: "uint64.attr", Value: uint64(100)},
			{Name: "uint32.attr", Value: uint32(32)},
			{Name: "uint16.attr", Value: uint16(16)},
			{Name: "uint8.attr", Value: uint8(8)},
			{Name: "float64.attr", Value: 3.14},
			{Name: "float32.attr", Value: float32(2.71)},
			{Name: "string.attr", Value: "test"},
		}
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		var f otelconfFactory
		res, err := f.createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.Equal(t, true, raw["bool.attr"])
		assert.Equal(t, int64(42), raw["int64.attr"])
		assert.Equal(t, int64(-32), raw["int32.attr"])
		assert.Equal(t, int64(-16), raw["int16.attr"])
		assert.Equal(t, int64(-8), raw["int8.attr"])
		assert.Equal(t, int64(123), raw["int.attr"])
		// uint and uint64 may not fit in OTLP's int64, so the Go SDK systematically converts them to strings
		assert.Equal(t, "200", raw["uint.attr"])
		assert.Equal(t, "100", raw["uint64.attr"])
		assert.Equal(t, int64(32), raw["uint32.attr"])
		assert.Equal(t, int64(16), raw["uint16.attr"])
		assert.Equal(t, int64(8), raw["uint8.attr"])
		assert.InDelta(t, 3.14, raw["float64.attr"], 0.001)
		assert.InDelta(t, 2.71, raw["float32.attr"], 0.01)
		assert.Equal(t, "test", raw["string.attr"])
	})
	t.Run("with unsupported type", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		cfg.Resource.Attributes = []config.AttributeNameValue{
			{Name: "complex.attr", Value: complex(1, 2)},
		}
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		var f otelconfFactory
		res, err := f.createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.Equal(t, "(1+2i)", raw["complex.attr"])
	})
}

func TestCreateFullResource(t *testing.T) {
	t.Run("empty config defaults", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config).Resource
		fullRes, err := createFullResource(t.Context(), component.BuildInfo{Command: "cmd", Version: "1.0.0"}, &cfg)
		require.NoError(t, err)
		assert.NotNil(t, fullRes.providerConfig.SchemaUrl)
		assert.NotEmpty(t, fullRes.providerConfig.Attributes)
	})

	t.Run("legacy removed default", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config).Resource
		legacy := confmap.NewFromStringMap(map[string]any{
			"service.name":        nil,
			"service.version":     nil,
			"service.instance.id": nil,
		})
		require.NoError(t, legacy.Unmarshal(&cfg))
		fullRes, err := createFullResource(t.Context(), component.BuildInfo{Command: "cmd", Version: "1.0.0"}, &cfg)
		require.NoError(t, err)
		for _, attr := range fullRes.providerConfig.Attributes {
			assert.NotContains(t, []string{"service.name", "service.version", "service.instance.id"}, attr.Name)
		}
	})

	t.Run("default values error", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config).Resource

		orig := defaultAttributeValues
		t.Cleanup(func() { defaultAttributeValues = orig })
		defaultAttributeValues = func(component.BuildInfo) (map[string]string, error) {
			return nil, assert.AnError
		}

		_, err := createFullResource(t.Context(), component.BuildInfo{Command: "cmd", Version: "1.0.0"}, &cfg)
		require.ErrorContains(t, err, assert.AnError.Error())
	})
}

func TestResourceConfigValidateAttributesListUnsupported(t *testing.T) {
	cfg := migration.ResourceConfigV030{}
	conf := confmap.NewFromStringMap(map[string]any{
		"attributes_list": "service.name=override",
	})
	require.NoError(t, conf.Unmarshal(&cfg))
	err := xconfmap.Validate(&cfg)
	require.ErrorContains(t, err, "resource::attributes_list is not currently supported")
}
