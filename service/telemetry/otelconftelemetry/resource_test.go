// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/internal/resource"
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
		cfg.Resource.Attributes = []resource.Attribute{
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
	t.Run("with detectors", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		cfg.Resource.Detection = &resource.DetectionConfig{
			Detectors: []any{"env", "host"},
		}
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		res, err := createResource(t.Context(), set, cfg)
		require.NoError(t, err)

		raw := res.Attributes().AsRaw()
		assert.Contains(t, raw, "service.instance.id")
		assert.Contains(t, raw, "service.name")
		assert.Contains(t, raw, "service.version")
	})
	t.Run("with invalid detector returns error", func(t *testing.T) {
		cfg := createDefaultConfig().(*Config)
		cfg.Resource.Detection = &resource.DetectionConfig{
			Detectors: []any{"invalid-detector"},
		}
		set := telemetry.Settings{BuildInfo: component.BuildInfo{Command: "otelcol", Version: "latest"}}
		_, err := createResource(t.Context(), set, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create resource")
		assert.Contains(t, err.Error(), "unknown detector")
	})
}
