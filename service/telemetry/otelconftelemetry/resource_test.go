// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	xotelconf "go.opentelemetry.io/contrib/otelconf/x"
	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/service/telemetry"
	"go.opentelemetry.io/collector/service/telemetry/otelconftelemetry/internal/migration"
)

func TestPutSDKAttribute(t *testing.T) {
	attrs := pcommon.NewMap()
	putSDKAttribute(attrs, "bool", attribute.BoolValue(true))
	putSDKAttribute(attrs, "int", attribute.Int64Value(7))
	putSDKAttribute(attrs, "float", attribute.Float64Value(1.5))
	putSDKAttribute(attrs, "str", attribute.StringValue("s"))
	putSDKAttribute(attrs, "bool.slice", attribute.BoolSliceValue([]bool{true, false}))
	putSDKAttribute(attrs, "int.slice", attribute.Int64SliceValue([]int64{1, 2}))
	putSDKAttribute(attrs, "float.slice", attribute.Float64SliceValue([]float64{1.5, 2.5}))
	putSDKAttribute(attrs, "str.slice", attribute.StringSliceValue([]string{"a", "b"}))
	putSDKAttribute(attrs, "invalid", attribute.Value{})

	assert.Equal(t, map[string]any{
		"bool":        true,
		"int":         int64(7),
		"float":       1.5,
		"str":         "s",
		"bool.slice":  []any{true, false},
		"int.slice":   []any{int64(1), int64(2)},
		"float.slice": []any{1.5, 2.5},
		"str.slice":   []any{"a", "b"},
		"invalid":     nil,
	}, attrs.AsRaw())
}

func TestCreateResource(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		res, _, err := createResource(t.Context(), set, cfg)
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
		res, _, err := createResource(t.Context(), set, cfg)
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
		res, _, err := createResource(t.Context(), set, cfg)
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
		res, _, err := createResource(t.Context(), set, cfg)
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
		res, schemaURL, err := createResource(t.Context(), set, cfg)
		require.NoError(t, err)
		assert.Equal(t, customSchemaURL, schemaURL)

		raw := res.Attributes().AsRaw()
		assert.Contains(t, raw, "service.name")
		assert.Equal(t, "test-service", raw["service.name"])
		assert.Contains(t, raw, "service.instance.id")
		assert.Contains(t, raw, "service.version")
	})
	t.Run("with host detector", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		cfg.Resource.DetectionDevelopment = &xotelconf.ExperimentalResourceDetection{
			Detectors: []xotelconf.ExperimentalResourceDetector{
				{Host: xotelconf.ExperimentalHostResourceDetector{}},
			},
		}
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		res, _, err := createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.Contains(t, raw, "host.name")
		assert.Contains(t, raw, "os.type")
		assert.Contains(t, raw, "os.description")
	})
	t.Run("with process detector (slice-valued attribute)", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		cfg.Resource.DetectionDevelopment = &xotelconf.ExperimentalResourceDetection{
			Detectors: []xotelconf.ExperimentalResourceDetector{
				{Process: xotelconf.ExperimentalProcessResourceDetector{}},
			},
		}
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		res, _, err := createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		require.Contains(t, raw, "process.command_args")
		expected := make([]any, len(os.Args))
		for i, arg := range os.Args {
			expected[i] = arg
		}
		assert.Equal(t, expected, raw["process.command_args"])
	})
	t.Run("with service detector explicit attributes still win", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		cfg.Resource.DetectionDevelopment = &xotelconf.ExperimentalResourceDetection{
			Detectors: []xotelconf.ExperimentalResourceDetector{
				{Service: xotelconf.ExperimentalServiceResourceDetector{}},
			},
		}
		cfg.Resource.Attributes = []config.AttributeNameValue{
			{Name: "service.name", Value: "configured-service"},
		}
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		res, _, err := createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.Equal(t, "configured-service", raw["service.name"])
		assert.Contains(t, raw, "service.instance.id")
	})
	t.Run("service detector preserves collector defaults", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		cfg.Resource.DetectionDevelopment = &xotelconf.ExperimentalResourceDetection{
			Detectors: []xotelconf.ExperimentalResourceDetector{
				{Service: xotelconf.ExperimentalServiceResourceDetector{}},
			},
		}

		orig := defaultAttributeValues
		t.Cleanup(func() { defaultAttributeValues = orig })
		defaultAttributeValues = func(component.BuildInfo) (map[string]string, error) {
			return map[string]string{
				"service.name":        "collector-service",
				"service.version":     "1.2.3",
				"service.instance.id": "collector-instance",
			}, nil
		}

		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		res, _, err := createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.Equal(t, "collector-service", raw["service.name"])
		assert.Equal(t, "collector-instance", raw["service.instance.id"])
		assert.Equal(t, "1.2.3", raw["service.version"])
	})
	t.Run("stable resource detectors remain ignored", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		cfg.Resource.Detectors = &config.Detectors{
			Attributes: &config.DetectorsAttributes{
				Included: []string{"host.*", "os.*"},
			},
		}
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		res, _, err := createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.NotContains(t, raw, "host.name")
		assert.NotContains(t, raw, "os.type")
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
		res, _, err := createResource(t.Context(), set, cfg)
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
		res, _, err := createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.Equal(t, "(1+2i)", raw["complex.attr"])
	})
}

func TestCreateResource_DefaultAttributeValuesError(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}

	orig := defaultAttributeValues
	t.Cleanup(func() { defaultAttributeValues = orig })
	defaultAttributeValues = func(component.BuildInfo) (map[string]string, error) {
		return nil, assert.AnError
	}

	res, _, err := createResource(t.Context(), set, cfg)
	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, pcommon.Resource{}, res)
}

func TestCreateResource_ExperimentalSDKError(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Resource.DetectionDevelopment = &xotelconf.ExperimentalResourceDetection{}
	set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}

	orig := newExperimentalSDK
	t.Cleanup(func() { newExperimentalSDK = orig })
	newExperimentalSDK = func(...xotelconf.ConfigurationOption) (xotelconf.SDK, error) {
		return xotelconf.SDK{}, assert.AnError
	}

	res, _, err := createResource(t.Context(), set, cfg)
	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, pcommon.Resource{}, res)
}

func TestDefaultAttributeValues(t *testing.T) {
	buildInfo := component.BuildInfo{
		Command: "otelcol",
		Version: "1.0.0",
	}

	t.Run("defaults included", func(t *testing.T) {
		defaults, err := defaultAttributeValues(buildInfo)
		require.NoError(t, err)
		assert.Equal(t, buildInfo.Command, defaults["service.name"])
		assert.Equal(t, buildInfo.Version, defaults["service.version"])
		_, ok := defaults["service.instance.id"]
		assert.True(t, ok)
	})

	t.Run("uuid failure", func(t *testing.T) {
		orig := defaultAttributeValues
		t.Cleanup(func() { defaultAttributeValues = orig })
		defaultAttributeValues = func(component.BuildInfo) (map[string]string, error) {
			return nil, assert.AnError
		}

		_, err := defaultAttributeValues(buildInfo)
		require.ErrorContains(t, err, assert.AnError.Error())
	})
}

func TestCreateInitialResourceConfig(t *testing.T) {
	t.Run("empty config defaults", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config).Resource
		resourceConfig, err := createInitialResourceConfig(component.BuildInfo{Command: "cmd", Version: "1.0.0"}, &cfg)
		require.NoError(t, err)
		require.NotNil(t, resourceConfig.SchemaUrl)
		assert.Equal(t, *cfg.SchemaUrl, *resourceConfig.SchemaUrl)
		assert.Len(t, resourceConfig.Attributes, 3)
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

func TestCreateFixedResourceConfig(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}

	res, schemaURL, err := createResource(t.Context(), set, cfg)
	require.NoError(t, err)

	resourceConfig, err := createFixedResourceConfig(&cfg.Resource, &res, schemaURL)
	require.NoError(t, err)
	require.NotNil(t, resourceConfig.SchemaUrl)
	assert.Equal(t, schemaURL, *resourceConfig.SchemaUrl)

	got := make(map[string]any, len(resourceConfig.Attributes))
	for _, attr := range resourceConfig.Attributes {
		got[attr.Name] = attr.Value
	}
	assert.Equal(t, "otelcol", got["service.name"])
	assert.Equal(t, "latest", got["service.version"])
	assert.Contains(t, got, "service.instance.id")

	t.Run("missing resource errors", func(t *testing.T) {
		_, err := createFixedResourceConfig(&cfg.Resource, nil, "")
		require.ErrorIs(t, err, errMissingCollectorResource)
	})
}

func TestFactoryDoesNotCacheResourceAcrossConfigs(t *testing.T) {
	factory := NewFactory()

	cfg1 := createDefaultConfig().(*Config)
	cfg1.Resource.Attributes = []config.AttributeNameValue{{Name: "service.name", Value: "svc-1"}}

	cfg2 := createDefaultConfig().(*Config)
	cfg2.Resource.Attributes = []config.AttributeNameValue{{Name: "service.name", Value: "svc-2"}}

	res1, _, err := factory.CreateResource(t.Context(), telemetry.Settings{
		BuildInfo: component.BuildInfo{Command: "otelcol", Version: "1.0.0"},
	}, cfg1)
	require.NoError(t, err)

	res2, _, err := factory.CreateResource(t.Context(), telemetry.Settings{
		BuildInfo: component.BuildInfo{Command: "otelcol", Version: "2.0.0"},
	}, cfg2)
	require.NoError(t, err)

	assert.Equal(t, "svc-1", res1.Attributes().AsRaw()["service.name"])
	assert.Equal(t, "svc-2", res2.Attributes().AsRaw()["service.name"])
}
