// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/service/telemetry"
	"go.opentelemetry.io/collector/service/telemetry/otelconftelemetry/internal/migration"
)

func TestCreateResource(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		res, err := createResource(t.Context(), set, cfg)
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
		require.NoError(t, cfg.Resource.Unmarshal(legacy))
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		res, err := createResource(t.Context(), set, cfg)
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
		require.NoError(t, cfg.Resource.Unmarshal(legacy))
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		res, err := createResource(t.Context(), set, cfg)
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
		res, err := createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.Equal(t, map[string]any{
			"extra.attr":      "value",
			"service.name":    "custom-service",
			"service.version": "0.1.0",
		}, raw)
	})
	t.Run("with attributes_list", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		list := "service.version=1.2.3,custom.attr=foo"
		cfg.Resource.AttributesList = &list
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		res, err := createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.Equal(t, "1.2.3", raw["service.version"])
		assert.Equal(t, "foo", raw["custom.attr"])
		assert.Equal(t, "otelcol", raw["service.name"])
		assert.Contains(t, raw, "service.instance.id")
	})
	t.Run("with attributes_list invalid pair", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		list := "invalidpair"
		cfg.Resource.AttributesList = &list
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		_, err := createResource(t.Context(), set, cfg)
		require.ErrorContains(t, err, "resource attributes_list has missing value")
	})
	t.Run("with attributes_list empty name", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		list := "=value"
		cfg.Resource.AttributesList = &list
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		_, err := createResource(t.Context(), set, cfg)
		require.ErrorContains(t, err, "resource attribute is missing name")
	})
	t.Run("with attributes_list invalid escape", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		list := "custom.attr=%ZZ"
		cfg.Resource.AttributesList = &list
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		res, err := createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.Equal(t, "%ZZ", raw["custom.attr"])
	})
	t.Run("with attributes_list empty", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		list := "   "
		cfg.Resource.AttributesList = &list
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		res, err := createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.Contains(t, raw, "service.name")
		assert.Contains(t, raw, "service.version")
		assert.Contains(t, raw, "service.instance.id")
	})
	t.Run("with custom schema_url", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		customSchemaURL := "https://opentelemetry.io/schemas/1.39.0"
		cfg.Resource.SchemaUrl = &customSchemaURL
		cfg.Resource.Attributes = []config.AttributeNameValue{
			{Name: "service.name", Value: "test-service"},
		}
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		res, err := createResource(t.Context(), set, cfg)
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
		res, err := createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.Contains(t, raw, "service.name")
		assert.Equal(t, "test-service", raw["service.name"])
	})
	t.Run("with typed attributes", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		var largeUint uint
		if ^uint(0) > uint(9223372036854775807) {
			largeUint = uint(9223372036854775808)
		} else {
			largeUint = uint(200)
		}
		cfg.Resource.Attributes = []config.AttributeNameValue{
			{Name: "bool.attr", Value: true},
			{Name: "int64.attr", Value: int64(42)},
			{Name: "int32.attr", Value: int32(-32)},
			{Name: "int16.attr", Value: int16(-16)},
			{Name: "int8.attr", Value: int8(-8)},
			{Name: "int.attr", Value: int(123)},
			{Name: "uint.small", Value: uint(200)},
			{Name: "uint64.small", Value: uint64(100)},
			{Name: "uint64.large", Value: uint64(9223372036854775808)},
			{Name: "uint32.attr", Value: uint32(32)},
			{Name: "uint16.attr", Value: uint16(16)},
			{Name: "uint8.attr", Value: uint8(8)},
			{Name: "uint.attr", Value: largeUint},
			{Name: "float64.attr", Value: 3.14},
			{Name: "float32.attr", Value: float32(2.71)},
			{Name: "string.attr", Value: "test"},
		}
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		res, err := createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.Equal(t, true, raw["bool.attr"])
		assert.Equal(t, int64(42), raw["int64.attr"])
		assert.Equal(t, int64(-32), raw["int32.attr"])
		assert.Equal(t, int64(-16), raw["int16.attr"])
		assert.Equal(t, int64(-8), raw["int8.attr"])
		assert.Equal(t, int64(123), raw["int.attr"])
		assert.Equal(t, int64(100), raw["uint64.small"])
		assert.Equal(t, "9223372036854775808", raw["uint64.large"])
		assert.Equal(t, int64(32), raw["uint32.attr"])
		assert.Equal(t, int64(16), raw["uint16.attr"])
		assert.Equal(t, int64(8), raw["uint8.attr"])
		assert.Equal(t, int64(200), raw["uint.small"])
		if largeUint > uint(9223372036854775807) {
			assert.Equal(t, "9223372036854775808", raw["uint.attr"])
		} else {
			assert.Equal(t, int64(200), raw["uint.attr"])
		}
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
		res, err := createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.Equal(t, "(1+2i)", raw["complex.attr"])
	})
	t.Run("with empty attribute name", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		cfg.Resource.Attributes = []config.AttributeNameValue{
			{Name: "", Value: "value"},
		}
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		_, err := createResource(t.Context(), set, cfg)
		require.ErrorContains(t, err, "resource attribute is missing name")
	})
}

func TestResourceConfigWithDefaults(t *testing.T) {
	t.Run("nil config defaults", func(t *testing.T) {
		resCfg, err := resourceConfigWithDefaults(component.BuildInfo{Command: "cmd", Version: "1.0.0"}, nil)
		require.NoError(t, err)
		assert.NotNil(t, resCfg.SchemaUrl)
		assert.NotEmpty(t, resCfg.Attributes)
	})

	t.Run("attributes_list prevents default", func(t *testing.T) {
		cfg := migration.ResourceConfigV030{}
		list := "service.name=override"
		cfg.AttributesList = &list
		resCfg, err := resourceConfigWithDefaults(component.BuildInfo{Command: "cmd", Version: "1.0.0"}, &cfg)
		require.NoError(t, err)

		for _, attr := range resCfg.Attributes {
			assert.NotEqual(t, "service.name", attr.Name)
		}
	})

	t.Run("explicitly removed default", func(t *testing.T) {
		cfg := migration.ResourceConfigV030{
			Resource: config.Resource{
				Attributes: []config.AttributeNameValue{
					{Name: "service.version", Value: nil},
				},
			},
		}
		resCfg, err := resourceConfigWithDefaults(component.BuildInfo{Command: "cmd", Version: "1.0.0"}, &cfg)
		require.NoError(t, err)

		for _, attr := range resCfg.Attributes {
			assert.NotEqual(t, "service.version", attr.Name)
		}
	})

	t.Run("legacy removed default", func(t *testing.T) {
		cfg := migration.ResourceConfigV030{}
		legacy := confmap.NewFromStringMap(map[string]any{
			"service.name":        nil,
			"service.version":     nil,
			"service.instance.id": nil,
		})
		require.NoError(t, cfg.Unmarshal(legacy))
		resCfg, err := resourceConfigWithDefaults(component.BuildInfo{Command: "cmd", Version: "1.0.0"}, &cfg)
		require.NoError(t, err)
		for _, attr := range resCfg.Attributes {
			assert.NotContains(t, []string{"service.name", "service.version", "service.instance.id"}, attr.Name)
		}
	})

	t.Run("default values error", func(t *testing.T) {
		orig := defaultAttributeValues
		t.Cleanup(func() { defaultAttributeValues = orig })
		defaultAttributeValues = func(component.BuildInfo, map[string]struct{}) (map[string]string, error) {
			return nil, assert.AnError
		}

		_, err := resourceConfigWithDefaults(component.BuildInfo{Command: "cmd", Version: "1.0.0"}, &migration.ResourceConfigV030{})
		require.ErrorContains(t, err, assert.AnError.Error())
	})
}

func TestPcommonValueToAttribute(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    pcommon.Value
		expected attribute.KeyValue
	}{
		{
			name:     "bool",
			key:      "test.bool",
			value:    pcommon.NewValueBool(true),
			expected: attribute.Bool("test.bool", true),
		},
		{
			name:     "int",
			key:      "test.int",
			value:    pcommon.NewValueInt(42),
			expected: attribute.Int64("test.int", 42),
		},
		{
			name:     "double",
			key:      "test.double",
			value:    pcommon.NewValueDouble(3.14),
			expected: attribute.Float64("test.double", 3.14),
		},
		{
			name:     "string",
			key:      "test.string",
			value:    pcommon.NewValueStr("hello"),
			expected: attribute.String("test.string", "hello"),
		},
		{
			name:     "bytes",
			key:      "test.bytes",
			value:    pcommon.NewValueBytes(),
			expected: attribute.String("test.bytes", ""),
		},
		{
			name:     "slice",
			key:      "test.slice",
			value:    pcommon.NewValueSlice(),
			expected: attribute.String("test.slice", "[]"),
		},
		{
			name:     "map",
			key:      "test.map",
			value:    pcommon.NewValueMap(),
			expected: attribute.String("test.map", "map[]"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pcommonValueToAttribute(tt.key, tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPcommonAttrsToOTelAttrs(t *testing.T) {
	t.Run("nil resource", func(t *testing.T) {
		result := pcommonAttrsToOTelAttrs(nil)
		assert.Empty(t, result)
	})

	t.Run("resource with various types", func(t *testing.T) {
		res := pcommon.NewResource()
		attrs := res.Attributes()
		attrs.PutBool("bool.key", true)
		attrs.PutInt("int.key", 42)
		attrs.PutDouble("double.key", 3.14)
		attrs.PutStr("string.key", "value")
		attrs.PutEmptyBytes("bytes.key").FromRaw([]byte("data"))

		result := pcommonAttrsToOTelAttrs(&res)
		assert.Len(t, result, 5)

		attrMap := make(map[string]attribute.Value)
		for _, kv := range result {
			attrMap[string(kv.Key)] = kv.Value
		}

		assert.Equal(t, attribute.BoolValue(true), attrMap["bool.key"])
		assert.Equal(t, attribute.Int64Value(42), attrMap["int.key"])
		assert.Equal(t, attribute.Float64Value(3.14), attrMap["double.key"])
		assert.Equal(t, attribute.StringValue("value"), attrMap["string.key"])
		assert.Equal(t, attribute.StringValue("data"), attrMap["bytes.key"])
	})
}
