// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
)

// newTestBuilder constructs a TelemetryBuilder backed by a fresh in-memory
// telemetry and registers cleanup. It returns the builder along with the
// telemetry so tests can read back the recorded metrics.
func newTestBuilder(t *testing.T, metricVariation string) (*TelemetryBuilder, *componenttest.Telemetry) {
	t.Helper()
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	tb, err := NewTelemetryBuilder(tt.NewTelemetrySettings(), metricVariation)
	require.NoError(t, err)
	t.Cleanup(tb.Shutdown)
	return tb, tt
}

// assertMetricName records a sample value and checks that the metric is
// retrievable under the expected variation-aware name.
func assertMetricName(t *testing.T, tb *TelemetryBuilder, tt *componenttest.Telemetry, name string) {
	t.Helper()
	tb.EnqueueFailedLogRecords.Add(t.Context(), 1)
	got, err := tt.GetMetric(name)
	require.NoError(t, err)
	assert.Equal(t, name, got.Name)
}

func TestMetricVariation(t *testing.T) {
	tests := []struct {
		name            string
		metricVariation string
		expected        string
	}{
		{
			name:            "exporter variation",
			metricVariation: "exporter",
			expected:        "otelcol_exporter_enqueue_failed_log_records",
		},
		{
			name:            "processor variation",
			metricVariation: "processor",
			expected:        "otelcol_processor_enqueue_failed_log_records",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tb, tt := newTestBuilder(t, tc.metricVariation)
			assertMetricName(t, tb, tt, tc.expected)
		})
	}
}

// TestMetricVariation_Required verifies that NewTelemetryBuilder rejects an
// empty metricVariation when allow_name_variation is set.
func TestMetricVariation_Required(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	_, err := NewTelemetryBuilder(tt.NewTelemetrySettings(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metricVariation is required")
}
