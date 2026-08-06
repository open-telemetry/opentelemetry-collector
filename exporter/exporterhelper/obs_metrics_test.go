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

// TestObsMetricsUnsetOperationsAreNoOps verifies that a config that implements
// only some operations does not panic on the others.
func TestObsMetricsUnsetOperationsAreNoOps(t *testing.T) {
	var recorded int64
	metrics := NewObsMetrics(ObsMetricsConfig{
		RecordSent: func(_ context.Context, items int64) {
			recorded = items
		},
	})

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

func TestObsMetricsOperationsAreForwarded(t *testing.T) {
	var calls []string
	observers := map[string]QueueObserver{}
	metrics := NewObsMetrics(ObsMetricsConfig{
		RecordEnqueueFailure: func(context.Context, int64) { calls = append(calls, "enqueue_failure") },
		RecordBatchSendSize:  func(context.Context, int64, int64) { calls = append(calls, "batch_send_size") },
		RegisterQueueSize: func(observe QueueObserver) error {
			observers["size"] = observe
			return nil
		},
		RegisterQueueCapacity: func(observe QueueObserver) error {
			observers["capacity"] = observe
			return nil
		},
		RecordInFlight:    func(context.Context, int64) { calls = append(calls, "in_flight") },
		RecordSent:        func(context.Context, int64) { calls = append(calls, "sent") },
		RecordSendFailure: func(context.Context, int64, ...metric.AddOption) { calls = append(calls, "send_failure") },
	})

	ctx := context.Background()
	metrics.RecordEnqueueFailure(ctx, 1)
	metrics.RecordBatchSendSize(ctx, 1, 2)
	metrics.RecordInFlight(ctx, 1)
	metrics.RecordSent(ctx, 1)
	metrics.RecordSendFailure(ctx, 1)
	require.Equal(t, []string{"enqueue_failure", "batch_send_size", "in_flight", "sent", "send_failure"}, calls)

	require.NoError(t, metrics.RegisterQueueSize(func() int64 { return 3 }))
	require.NoError(t, metrics.RegisterQueueCapacity(func() int64 { return 9 }))
	require.Equal(t, int64(3), observers["size"]())
	require.Equal(t, int64(9), observers["capacity"]())
}

// TestObsMetricsShutdownIsIdempotent covers the contract relied on by
// components that shut down their own metrics when exporter construction fails
// after exporterhelper already took ownership.
func TestObsMetricsShutdownIsIdempotent(t *testing.T) {
	shutdowns := 0
	metrics := NewObsMetrics(ObsMetricsConfig{Shutdown: func() { shutdowns++ }})

	metrics.Shutdown()
	metrics.Shutdown()
	metrics.Shutdown()
	require.Equal(t, 1, shutdowns)
}

func TestWithObsMetricsNil(t *testing.T) {
	// A nil *ObsMetrics must be rejected rather than stored as a non-nil
	// interface value holding a nil pointer.
	err := WithObsMetrics(nil)(&internal.BaseExporter{})
	require.ErrorContains(t, err, "must not be nil")
}
