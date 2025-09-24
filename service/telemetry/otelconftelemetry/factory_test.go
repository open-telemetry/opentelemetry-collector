// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/featuregate"
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
			prev := useLocalHostAsDefaultMetricsAddressFeatureGate.IsEnabled()
			require.NoError(t, featuregate.GlobalRegistry().Set(useLocalHostAsDefaultMetricsAddressFeatureGate.ID(), tt.gate))
			defer func() {
				// Restore previous value.
				require.NoError(t, featuregate.GlobalRegistry().Set(useLocalHostAsDefaultMetricsAddressFeatureGate.ID(), prev))
			}()
			cfg := NewFactory().CreateDefaultConfig()
			require.Len(t, cfg.(*Config).Metrics.Readers, 1)
			assert.Equal(t, tt.expected, *cfg.(*Config).Metrics.Readers[0].Pull.Exporter.Prometheus.Host)
			assert.Equal(t, 8888, *cfg.(*Config).Metrics.Readers[0].Pull.Exporter.Prometheus.Port)
		})
	}
}
