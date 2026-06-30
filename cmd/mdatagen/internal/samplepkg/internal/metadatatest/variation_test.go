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

// TestAssertHelpers_Variation verifies that the AssertEqual* helpers take a
// variation argument and look up metrics under the variation-aware name.
func TestAssertHelpers_Variation(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	tb, err := metadata.NewTelemetryBuilder(tt.NewTelemetrySettings(), "processor")
	require.NoError(t, err)
	t.Cleanup(tb.Shutdown)

	tb.EnqueueFailedLogRecords.Add(context.Background(), 1)

	AssertEqualEnqueueFailedLogRecords(t, tt, "processor",
		[]metricdata.DataPoint[int64]{{Value: 1}},
		metricdatatest.IgnoreTimestamp(),
	)
}
