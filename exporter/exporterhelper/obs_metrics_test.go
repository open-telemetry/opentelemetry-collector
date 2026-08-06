// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
)

// TestFuncObsMetricsZeroValueIsNoOp verifies that the zero value implements
// every operation, so a component only sets the events it reports.
func TestFuncObsMetricsZeroValueIsNoOp(t *testing.T) {
	var recorded int64
	metrics := FuncObsMetrics{
		RecordSentFunc: func(_ context.Context, items int64) {
			recorded = items
		},
	}

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

func TestFuncObsMetricsOperationsAreForwarded(t *testing.T) {
	var calls []string
	observers := map[string]func() int64{}
	metrics := FuncObsMetrics{
		RecordEnqueueFailureFunc: func(context.Context, int64) { calls = append(calls, "enqueue_failure") },
		RecordBatchSendSizeFunc:  func(context.Context, int64, int64) { calls = append(calls, "batch_send_size") },
		RegisterQueueSizeFunc: func(observeSize func() int64) error {
			observers["size"] = observeSize
			return nil
		},
		RegisterQueueCapacityFunc: func(observeCapacity func() int64) error {
			observers["capacity"] = observeCapacity
			return nil
		},
		RecordInFlightFunc:     func(context.Context, int64) { calls = append(calls, "in_flight") },
		RecordSentFunc:         func(context.Context, int64) { calls = append(calls, "sent") },
		RecordSendFailureFunc:  func(context.Context, int64, ...metric.AddOption) { calls = append(calls, "send_failure") },
		ShutdownObsMetricsFunc: func() { calls = append(calls, "shutdown") },
	}

	ctx := context.Background()
	metrics.RecordEnqueueFailure(ctx, 1)
	metrics.RecordBatchSendSize(ctx, 1, 2)
	metrics.RecordInFlight(ctx, 1)
	metrics.RecordSent(ctx, 1)
	metrics.RecordSendFailure(ctx, 1)
	metrics.Shutdown()
	require.Equal(t, []string{"enqueue_failure", "batch_send_size", "in_flight", "sent", "send_failure", "shutdown"}, calls)

	require.NoError(t, metrics.RegisterQueueSize(func() int64 { return 3 }))
	require.NoError(t, metrics.RegisterQueueCapacity(func() int64 { return 9 }))
	require.Equal(t, int64(3), observers["size"]())
	require.Equal(t, int64(9), observers["capacity"]())
}

func TestWithObsMetricsNil(t *testing.T) {
	err := WithObsMetrics(nil)(&internal.BaseExporter{})
	require.ErrorContains(t, err, "must not be nil")
}
