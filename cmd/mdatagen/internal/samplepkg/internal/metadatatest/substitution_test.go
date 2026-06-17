// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metadatatest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal/samplepkg/internal/metadata"
	"go.opentelemetry.io/collector/component/componenttest"
)

// TestAssertHelpers_WithMetricNamePrefix verifies that the AssertEqual* helpers
// accept WithMetricNamePrefix and look up metrics under their fully prefixed
// names.
func TestAssertHelpers_WithMetricNamePrefix(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	tb, err := metadata.NewTelemetryBuilder(
		tt.NewTelemetrySettings(),
		metadata.WithMetricNamePrefix("otelcol_processor_"),
	)
	require.NoError(t, err)
	t.Cleanup(tb.Shutdown)

	tb.EnqueueFailedLogRecords.Add(context.Background(), 1)

	AssertEqualEnqueueFailedLogRecords(t, tt,
		[]metricdata.DataPoint[int64]{{Value: 1}},
		WithMetricNamePrefix("otelcol_processor_"),
		metricdatatest.IgnoreTimestamp(),
	)
}
