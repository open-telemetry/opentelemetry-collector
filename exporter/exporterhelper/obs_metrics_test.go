// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestObsMetricsFunctions(t *testing.T) {
	var recorded int64
	metrics := NewObsMetrics(
		WithRecordSent(func(_ context.Context, items int64) {
			recorded = items
		}),
	)

	metrics.RecordSent(context.Background(), 7)
	require.Equal(t, int64(7), recorded)
	require.NoError(t, metrics.RegisterQueueSize(nil))
	require.NoError(t, metrics.RegisterQueueCapacity(nil))
	metrics.RecordEnqueueFailure(context.Background(), 1)
	metrics.RecordBatchSendSize(context.Background(), 1, 2)
	metrics.RecordInFlight(context.Background(), 1)
	metrics.RecordSendFailure(context.Background(), 1)
	metrics.Shutdown()
}

func TestObserveQueueFunc(t *testing.T) {
	require.Zero(t, ObserveQueueFunc(nil).Observe())
	require.Equal(t, int64(5), ObserveQueueFunc(func() int64 { return 5 }).Observe())
}
