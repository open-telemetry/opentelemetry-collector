// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/telemetry"
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
	t.Run("with resource attributes", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		cfg.Resource = map[string]*string{
			"extra.attr":          ptr("value"),
			"service.name":        ptr("custom-service"),
			"service.version":     ptr("0.1.0"),
			"service.instance.id": nil, // removes the attribute
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
}

func TestAttributeValueString(t *testing.T) {
	t.Run("string attribute", func(t *testing.T) {
		val := attribute.StringValue("test-value")
		result, err := attributeValueString("test.attr", val)
		require.NoError(t, err)
		assert.Equal(t, "test-value", result)
	})

	t.Run("int64 attribute returns error", func(t *testing.T) {
		val := attribute.Int64Value(42)
		result, err := attributeValueString("test.attr", val)
		require.Error(t, err)
		assert.Empty(t, result)
		assert.Contains(t, err.Error(), `attribute "test.attr": expected string, got INT64`)
	})

	t.Run("bool attribute returns error", func(t *testing.T) {
		val := attribute.BoolValue(true)
		result, err := attributeValueString("test.attr", val)
		require.Error(t, err)
		assert.Empty(t, result)
		assert.Contains(t, err.Error(), `attribute "test.attr": expected string, got BOOL`)
	})

	t.Run("float64 attribute returns error", func(t *testing.T) {
		val := attribute.Float64Value(3.14)
		result, err := attributeValueString("test.attr", val)
		require.Error(t, err)
		assert.Empty(t, result)
		assert.Contains(t, err.Error(), `attribute "test.attr": expected string, got FLOAT64`)
	})
}
