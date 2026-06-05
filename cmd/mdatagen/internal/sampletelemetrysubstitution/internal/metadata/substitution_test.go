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
// retrievable under the expected substituted name.
func assertMetricName(t *testing.T, tb *TelemetryBuilder, tt *componenttest.Telemetry, name string) {
	t.Helper()
	tb.ExporterEnqueueFailedLogRecords.Add(t.Context(), 1)
	got, err := tt.GetMetric(name)
	require.NoError(t, err)
	assert.Equal(t, name, got.Name)
}

func TestNameSubstitution(t *testing.T) {
	tests := []struct {
		name     string
		opts     []TelemetryBuilderOption
		expected string
	}{
		{
			name:     "default emits otelcol_exporter_*",
			expected: "otelcol_exporter_enqueue_failed_log_records",
		},
		{
			name:     "processor replacement emits otelcol_processor_*",
			opts:     []TelemetryBuilderOption{WithMetricNamePrefixReplacement("otelcol_exporter_", "otelcol_processor_")},
			expected: "otelcol_processor_enqueue_failed_log_records",
		},
		{
			name:     "identity replacement is a no-op",
			opts:     []TelemetryBuilderOption{WithMetricNamePrefixReplacement("otelcol_exporter_", "otelcol_exporter_")},
			expected: "otelcol_exporter_enqueue_failed_log_records",
		},
		{
			name:     "empty newPrefix strips the configured oldPrefix",
			opts:     []TelemetryBuilderOption{WithMetricNamePrefixReplacement("otelcol_exporter_", "")},
			expected: "enqueue_failed_log_records",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tb, tt := newTestBuilder(t, tc.opts...)
			assertMetricName(t, tb, tt, tc.expected)
		})
	}
}

func TestNameSubstitution_Errors(t *testing.T) {
	tests := []struct {
		name     string
		opts     []TelemetryBuilderOption
		contains []string
	}{
		{
			name:     "empty oldPrefix is rejected",
			opts:     []TelemetryBuilderOption{WithMetricNamePrefixReplacement("", "otelcol_processor_")},
			contains: []string{"oldPrefix must not be empty"},
		},
		{
			name: "prefix mismatch reports every metric",
			opts: []TelemetryBuilderOption{WithMetricNamePrefixReplacement("not_a_real_prefix_", "x_")},
			contains: []string{
				`does not start with prefix "not_a_real_prefix_"`,
				"otelcol_exporter_enqueue_failed_log_records",
				"otelcol_exporter_queue_batch_send_size",
				"otelcol_exporter_queue_size",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tt := componenttest.NewTelemetry()
			t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

			_, err := NewTelemetryBuilder(tt.NewTelemetrySettings(), tc.opts...)
			require.Error(t, err)
			for _, want := range tc.contains {
				assert.Contains(t, err.Error(), want)
			}
		})
	}
}

