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

// Every operation is optional, so a component supplies only the ones it reports.
func TestObsMetricsNilOperationsAreNoOps(t *testing.T) {
	metrics := NewObsMetrics(nil, nil, nil, nil, nil)

	ctx := context.Background()
	require.NoError(t, metrics.RegisterQueueSize(nil))
	require.NoError(t, metrics.RegisterQueueCapacity(nil))
	metrics.RecordEnqueueFailure(ctx, 1)
	metrics.RecordEnqueueSize(ctx, 1, failBytesSize(t))
	metrics.RecordBatchSendSize(ctx, 1, failBytesSize(t))
	metrics.RecordInFlight(ctx, 1)
	metrics.RecordSent(ctx, 1)
	metrics.RecordSendFailure(ctx, 1)
	metrics.Shutdown()
}

func TestObsMetricsOperationsAreForwarded(t *testing.T) {
	var calls []string
	observers := map[string]func() int64{}

	metrics := NewObsMetrics(
		NewQueueBatchMetrics(
			NewQueueMetrics(
				func(context.Context, int64) { calls = append(calls, "enqueue_failure") },
				func(context.Context, int64, func() int64) { calls = append(calls, "enqueue_size") },
				func(observeSize func() int64) error {
					observers["size"] = observeSize
					return nil
				},
				func(observeCapacity func() int64) error {
					observers["capacity"] = observeCapacity
					return nil
				},
			),
			func(context.Context, int64, func() int64) { calls = append(calls, "batch_send_size") },
		),
		func(context.Context, int64) { calls = append(calls, "in_flight") },
		func(context.Context, int64) { calls = append(calls, "sent") },
		func(context.Context, int64, ...metric.AddOption) { calls = append(calls, "send_failure") },
		func() { calls = append(calls, "shutdown") },
	)

	ctx := context.Background()
	metrics.RecordEnqueueFailure(ctx, 1)
	metrics.RecordEnqueueSize(ctx, 1, nil)
	metrics.RecordBatchSendSize(ctx, 1, nil)
	metrics.RecordInFlight(ctx, 1)
	metrics.RecordSent(ctx, 1)
	metrics.RecordSendFailure(ctx, 1)
	metrics.Shutdown()
	require.Equal(t, []string{
		"enqueue_failure", "enqueue_size", "batch_send_size", "in_flight", "sent", "send_failure", "shutdown",
	}, calls)

	require.NoError(t, metrics.RegisterQueueSize(func() int64 { return 3 }))
	require.NoError(t, metrics.RegisterQueueCapacity(func() int64 { return 9 }))
	require.Equal(t, int64(3), observers["size"]())
	require.Equal(t, int64(9), observers["capacity"]())
}

// The queue and batch operations are optional as a group.
func TestObsMetricsWithoutQueueBatchMetrics(t *testing.T) {
	var sent int64
	metrics := NewObsMetrics(nil, nil, func(_ context.Context, items int64) { sent = items }, nil, nil)

	metrics.RecordEnqueueFailure(context.Background(), 1)
	metrics.RecordSent(context.Background(), 7)
	require.Equal(t, int64(7), sent)
}

// failBytesSize fails the test when a no-op operation measures the request.
func failBytesSize(t *testing.T) func() int64 {
	return func() int64 {
		t.Error("bytes size must not be measured when the operation reports nothing")
		return 0
	}
}

func TestWithObsMetricsNil(t *testing.T) {
	err := WithObsMetrics(nil)(&internal.BaseExporter{})
	require.ErrorContains(t, err, "must not be nil")
}
