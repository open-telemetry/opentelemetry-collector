// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package otelconftelemetry

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/service/internal/metadata"
)

func TestDefaultConfig(t *testing.T) {
	tests := []struct {
		expected string
		gate     bool
	}{
		{
			expected: "localhost",
			gate:     true,
		},
		{
			expected: "",
			gate:     false,
		},
	}

	for _, tt := range tests {
		t.Run("UseLocalHostAsDefaultMetricsAddress/"+strconv.FormatBool(tt.gate), func(t *testing.T) {
			prevNoReader := metadata.TelemetryNoDefaultMetricsReaderFeatureGate.IsEnabled()
			require.NoError(t, featuregate.GlobalRegistry().Set(metadata.TelemetryNoDefaultMetricsReaderFeatureGate.ID(), false))
			defer func() {
				require.NoError(t, featuregate.GlobalRegistry().Set(metadata.TelemetryNoDefaultMetricsReaderFeatureGate.ID(), prevNoReader))
			}()

			prev := metadata.TelemetryUseLocalHostAsDefaultMetricsAddressFeatureGate.IsEnabled()
			require.NoError(t, featuregate.GlobalRegistry().Set(metadata.TelemetryUseLocalHostAsDefaultMetricsAddressFeatureGate.ID(), tt.gate))
			defer func() {
				require.NoError(t, featuregate.GlobalRegistry().Set(metadata.TelemetryUseLocalHostAsDefaultMetricsAddressFeatureGate.ID(), prev))
			}()

			cfg := NewFactory().CreateDefaultConfig()
			require.Len(t, cfg.(*Config).Metrics.Readers, 1)
			assert.Equal(t, tt.expected, *cfg.(*Config).Metrics.Readers[0].Pull.Exporter.Prometheus.Host)
			assert.Equal(t, 8888, *cfg.(*Config).Metrics.Readers[0].Pull.Exporter.Prometheus.Port)
		})
	}
}

func TestDefaultConfigNoDefaultMetricsReader(t *testing.T) {
	prev := metadata.TelemetryNoDefaultMetricsReaderFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(metadata.TelemetryNoDefaultMetricsReaderFeatureGate.ID(), true))
	defer func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(metadata.TelemetryNoDefaultMetricsReaderFeatureGate.ID(), prev))
	}()

	cfg := NewFactory().CreateDefaultConfig()
	assert.Empty(t, cfg.(*Config).Metrics.Readers)
	assert.Equal(t, configtelemetry.LevelNone, cfg.(*Config).Metrics.Level)
}
