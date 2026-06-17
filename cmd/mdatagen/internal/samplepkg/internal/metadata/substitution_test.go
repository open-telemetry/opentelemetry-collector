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
func newTestBuilder(t *testing.T, opts ...TelemetryBuilderOption) (*TelemetryBuilder, *componenttest.Telemetry) {
	t.Helper()
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	tb, err := NewTelemetryBuilder(tt.NewTelemetrySettings(), opts...)
	require.NoError(t, err)
	t.Cleanup(tb.Shutdown)
	return tb, tt
}

// assertMetricName records a sample value and checks that the metric is
// retrievable under the expected prefixed name.
func assertMetricName(t *testing.T, tb *TelemetryBuilder, tt *componenttest.Telemetry, name string) {
	t.Helper()
	tb.EnqueueFailedLogRecords.Add(t.Context(), 1)
	got, err := tt.GetMetric(name)
	require.NoError(t, err)
	assert.Equal(t, name, got.Name)
}

func TestMetricNamePrefix(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		expected string
	}{
		{
			name:     "exporter prefix",
			prefix:   "otelcol_exporter_",
			expected: "otelcol_exporter_enqueue_failed_log_records",
		},
		{
			name:     "processor prefix",
			prefix:   "otelcol_processor_",
			expected: "otelcol_processor_enqueue_failed_log_records",
		},
		{
			name:     "bare prefix",
			prefix:   "x_",
			expected: "x_enqueue_failed_log_records",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tb, tt := newTestBuilder(t, WithMetricNamePrefix(tc.prefix))
			assertMetricName(t, tb, tt, tc.expected)
		})
	}
}

// TestMetricNamePrefix_Required verifies that NewTelemetryBuilder rejects a
// missing or empty WithMetricNamePrefix when allow_name_substitution is set.
func TestMetricNamePrefix_Required(t *testing.T) {
	tests := []struct {
		name string
		opts []TelemetryBuilderOption
	}{
		{name: "option omitted entirely"},
		{name: "empty prefix", opts: []TelemetryBuilderOption{WithMetricNamePrefix("")}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tt := componenttest.NewTelemetry()
			t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

			_, err := NewTelemetryBuilder(tt.NewTelemetrySettings(), tc.opts...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "WithMetricNamePrefix is required")
		})
	}
}
