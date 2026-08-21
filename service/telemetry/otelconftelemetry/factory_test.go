// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	tests := []struct {
		expected string
	}{
		{
			expected: "localhost",
		},
	}

	for _, tt := range tests {
		t.Run("UseLocalHostAsDefaultMetricsAddress", func(t *testing.T) {
			cfg := NewFactory().CreateDefaultConfig()
			require.Len(t, cfg.(*Config).Metrics.Readers, 1)
			assert.Equal(t, tt.expected, *cfg.(*Config).Metrics.Readers[0].Pull.Exporter.Prometheus.Host)
			assert.Equal(t, 8888, *cfg.(*Config).Metrics.Readers[0].Pull.Exporter.Prometheus.Port)
		})
	}
}
