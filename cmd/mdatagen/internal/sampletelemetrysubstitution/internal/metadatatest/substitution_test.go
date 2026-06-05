// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metadatatest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal/sampletelemetrysubstitution/internal/metadata"
	"go.opentelemetry.io/collector/component/componenttest"
)

// TestAssertHelpers_WithSubstitution verifies that the AssertEqual* helpers
// accept WithMetricNamePrefixReplacement and look up metrics under their
// substituted names.
func TestAssertHelpers_WithSubstitution(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	tb, err := metadata.NewTelemetryBuilder(
		tt.NewTelemetrySettings(),
		metadata.WithMetricNamePrefixReplacement("otelcol_exporter_", "otelcol_processor_"),
	)
	require.NoError(t, err)
	t.Cleanup(tb.Shutdown)

	tb.ExporterEnqueueFailedLogRecords.Add(context.Background(), 1)

	AssertEqualExporterEnqueueFailedLogRecords(t, tt,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		WithMetricNamePrefixReplacement("otelcol_exporter_", "otelcol_processor_"),
		metricdatatest.IgnoreTimestamp(),
	)
}
