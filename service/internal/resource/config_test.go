// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"

	"go.opentelemetry.io/collector/component"
)

var buildInfo = component.BuildInfo{
	Command: "otelcol",
	Version: "1.0.0",
}

func ptr[T any](v T) *T {
	return &v
}

func TestNew(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name      string
		cfg       *Config
		assertion func(t *testing.T, attrs map[string]any)
	}{
		{
			name: "default attributes",
			cfg:  &Config{},
			assertion: func(t *testing.T, attrs map[string]any) {
				require.Contains(t, attrs, "service.instance.id")
				instanceID, ok := attrs["service.instance.id"].(string)
				require.True(t, ok)
				_, err := uuid.Parse(instanceID)
				require.NoError(t, err)

				delete(attrs, "service.instance.id")
				assert.Equal(t, map[string]any{
					"service.name":    "otelcol",
					"service.version": "1.0.0",
				}, attrs)
			},
		},
		{
			name: "override defaults",
			cfg: &Config{
				Attributes: []Attribute{
					{Name: "service.name", Value: "my-service"},
					{Name: "service.version", Value: "1.2.3"},
					{Name: "service.instance.id", Value: "123"},
					{Name: "host.name", Value: "collector"},
				},
			},
			assertion: func(t *testing.T, attrs map[string]any) {
				assert.Equal(t, map[string]any{
					"service.name":        "my-service",
					"service.version":     "1.2.3",
					"service.instance.id": "123",
					"host.name":           "collector",
				}, attrs)
			},
		},
		{
			name: "remove defaults",
			cfg: &Config{
				Attributes: []Attribute{
					{Name: "service.name", Value: nil},
					{Name: "service.version", Value: nil},
					{Name: "service.instance.id", Value: nil},
				},
			},
			assertion: func(t *testing.T, attrs map[string]any) {
				assert.Equal(t, map[string]any{}, attrs)
			},
		},
		{
			name: "attributes_list lower priority",
			cfg: &Config{
				AttributesList: ptr("from.list=value,service.name=list-name"),
				Attributes: []Attribute{
					{Name: "service.name", Value: "config-name"},
					{Name: "deployment.environment", Value: "prod"},
				},
			},
			assertion: func(t *testing.T, attrs map[string]any) {
				require.Contains(t, attrs, "service.instance.id")
				delete(attrs, "service.instance.id")
				assert.Equal(t, map[string]any{
					"service.name":           "config-name",
					"service.version":        "1.0.0",
					"deployment.environment": "prod",
					"from.list":              "value",
				}, attrs)
			},
		},
		{
			name: "legacy inline attributes",
			cfg: &Config{
				DeprecatedAttributes: map[string]any{
					"service.name":        "legacy",
					"service.version":     "legacy-version",
					"service.instance.id": "legacy-id",
					"host.type":           "c6i",
				},
			},
			assertion: func(t *testing.T, attrs map[string]any) {
				assert.Equal(t, map[string]any{
					"service.name":        "legacy",
					"service.version":     "legacy-version",
					"service.instance.id": "legacy-id",
					"host.type":           "c6i",
				}, attrs)
			},
		},
		{
			name: "typed attributes",
			cfg: &Config{
				Attributes: []Attribute{
					{Name: "feature.enabled", Value: true},
					{Name: "retry.count", Value: 5},
					{Name: "latency.ms", Value: 12.5},
					{Name: "owners", Value: []any{"a", "b"}, Type: "string_array"},
					{Name: "success.flags", Value: []bool{true, false}, Type: "bool_array"},
					{Name: "limits", Value: []int{1, 2}, Type: "int_array"},
					{Name: "ratios", Value: []float64{0.1, 0.2}, Type: "double_array"},
				},
			},
			assertion: func(t *testing.T, attrs map[string]any) {
				require.Contains(t, attrs, "service.instance.id")
				delete(attrs, "service.instance.id")
				assert.Equal(t, map[string]any{
					"service.name":    "otelcol",
					"service.version": "1.0.0",
					"feature.enabled": true,
					"retry.count":     int64(5),
					"latency.ms":      12.5,
					"owners":          []string{"a", "b"},
					"success.flags":   []bool{true, false},
					"limits":          []int64{1, 2},
					"ratios":          []float64{0.1, 0.2},
				}, attrs)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, err := New(ctx, buildInfo, tt.cfg)
			require.NoError(t, err)
			tt.assertion(t, attributesAsRaw(res))
		})
	}
}

func TestInvalidConfigurations(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		cfg  *Config
	}{
		{
			name: "attributes_list invalid format",
			cfg: &Config{
				AttributesList: ptr("invalid-entry"),
			},
		},
		{
			name: "attribute missing name",
			cfg: &Config{
				Attributes: []Attribute{{Name: "", Value: "test"}},
			},
		},
		{
			name: "attribute invalid type",
			cfg: &Config{
				Attributes: []Attribute{{Name: "key", Value: true, Type: "string"}},
			},
		},
		{
			name: "legacy attribute non string",
			cfg: &Config{
				DeprecatedAttributes: map[string]any{"key": 1},
			},
		},
		{
			name: "detector entry missing name",
			cfg: &Config{
				Detection: &DetectionConfig{
					Detectors: []any{map[string]any{"env": nil, "host": nil}},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(ctx, buildInfo, tt.cfg)
			assert.Error(t, err)
		})
	}
}

func TestDetectors(t *testing.T) {
	ctx := context.Background()
	build := component.BuildInfo{
		Command: "otelcol-test",
		Version: "1.0.0",
	}

	t.Run("env detector", func(t *testing.T) {
		t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "test.key=test.value")
		cfg := &Config{
			Detection: &DetectionConfig{
				Detectors: []any{"env"},
			},
		}
		res, err := New(ctx, build, cfg)
		require.NoError(t, err)
		attrs := attributesAsRaw(res)
		assert.Equal(t, "test.value", attrs["test.key"])
		assert.Equal(t, "otelcol-test", attrs["service.name"])
	})

	t.Run("host detector", func(t *testing.T) {
		cfg := &Config{
			Detection: &DetectionConfig{
				Detectors: []any{"host"},
			},
		}
		res, err := New(ctx, build, cfg)
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.GreaterOrEqual(t, len(res.Attributes()), 3)
	})

	t.Run("multiple detectors", func(t *testing.T) {
		t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "env.attr=value")
		cfg := &Config{
			Detection: &DetectionConfig{
				Detectors: []any{"env", "host"},
			},
		}
		res, err := New(ctx, build, cfg)
		require.NoError(t, err)
		attrs := attributesAsRaw(res)
		assert.Equal(t, "value", attrs["env.attr"])
	})

	t.Run("override detector attribute", func(t *testing.T) {
		t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.name=from-env")
		cfg := &Config{
			Attributes: []Attribute{{Name: string(semconv.ServiceNameKey), Value: "from-config"}},
			Detection: &DetectionConfig{
				Detectors: []any{"env"},
			},
		}
		res, err := New(ctx, build, cfg)
		require.NoError(t, err)
		attrs := attributesAsRaw(res)
		assert.Equal(t, "from-config", attrs["service.name"])
	})

	t.Run("attribute filter include and exclude", func(t *testing.T) {
		t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "env.keep=value,env.secret=should-not-appear,other.attr=ignored")
		cfg := &Config{
			Detection: &DetectionConfig{
				Detectors: []any{"env"},
				Attributes: &IncludeExclude{
					Included: []string{"env.*"},
					Excluded: []string{"env.secret"},
				},
			},
		}
		res, err := New(ctx, build, cfg)
		require.NoError(t, err)
		attrs := attributesAsRaw(res)
		assert.Equal(t, "value", attrs["env.keep"])
		_, hasSecret := attrs["env.secret"]
		assert.False(t, hasSecret, "filtered attribute should not be present")
		_, hasOther := attrs["other.attr"]
		assert.False(t, hasOther, "non matching attribute should not be present")
	})

	t.Run("unknown detector returns error", func(t *testing.T) {
		cfg := &Config{
			Detection: &DetectionConfig{
				Detectors: []any{"invalid"},
			},
		}
		_, err := New(ctx, build, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown detector")
	})
}

func attributesAsRaw(res *sdkresource.Resource) map[string]any {
	raw := make(map[string]any)
	for _, kv := range res.Attributes() {
		raw[string(kv.Key)] = attributeValueToAny(kv.Value)
	}
	return raw
}

func attributeValueToAny(val attribute.Value) any {
	switch val.Type() {
	case attribute.STRING:
		return val.AsString()
	case attribute.BOOL:
		return val.AsBool()
	case attribute.INT64:
		return val.AsInt64()
	case attribute.FLOAT64:
		return val.AsFloat64()
	case attribute.STRINGSLICE:
		return append([]string(nil), val.AsStringSlice()...)
	case attribute.BOOLSLICE:
		return append([]bool(nil), val.AsBoolSlice()...)
	case attribute.INT64SLICE:
		return append([]int64(nil), val.AsInt64Slice()...)
	case attribute.FLOAT64SLICE:
		return append([]float64(nil), val.AsFloat64Slice()...)
	default:
		return nil
	}
}
