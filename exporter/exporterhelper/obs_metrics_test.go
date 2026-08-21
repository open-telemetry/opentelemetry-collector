// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
)

func TestNilValueFuncReturnsZero(t *testing.T) {
	require.Zero(t, NewInt64Value(nil).Value())
}

// Every operation is optional, so a component supplies only the ones it reports.
func TestObsMetricsNilOperationsAreNoOps(t *testing.T) {
	metrics := NewObsMetrics()

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
	observers := map[string]Int64Value{}

	metrics := NewObsMetrics(
		WithQueueBatchMetrics(NewQueueBatchMetrics(
			WithQueueMetrics(NewQueueMetrics(
				WithRecordEnqueueFailure(func(context.Context, int64) {
					calls = append(calls, "enqueue_failure")
				}),
				WithRecordEnqueueSize(func(context.Context, int64, Int64Value) {
					calls = append(calls, "enqueue_size")
				}),
				WithRegisterQueueSize(func(observeSize Int64Value) error {
					observers["size"] = observeSize
					return nil
				}),
				WithRegisterQueueCapacity(func(observeCapacity Int64Value) error {
					observers["capacity"] = observeCapacity
					return nil
				}),
			)),
			WithRecordBatchSendSize(func(context.Context, int64, Int64Value) {
				calls = append(calls, "batch_send_size")
			}),
		)),
		WithRecordInFlight(func(context.Context, int64) { calls = append(calls, "in_flight") }),
		WithRecordSent(func(context.Context, int64) { calls = append(calls, "sent") }),
		WithRecordSendFailure(func(context.Context, int64, ...metric.AddOption) {
			calls = append(calls, "send_failure")
		}),
		WithMetricsShutdown(func() { calls = append(calls, "shutdown") }),
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

	require.NoError(t, metrics.RegisterQueueSize(NewInt64Value(func() int64 { return 3 })))
	require.NoError(t, metrics.RegisterQueueCapacity(NewInt64Value(func() int64 { return 9 })))
	require.Equal(t, int64(3), observers["size"].Value())
	require.Equal(t, int64(9), observers["capacity"].Value())
}

// The queue and batch operations are optional as a group.
func TestObsMetricsWithoutQueueBatchMetrics(t *testing.T) {
	var sent int64
	metrics := NewObsMetrics(WithRecordSent(func(_ context.Context, items int64) { sent = items }))

	metrics.RecordEnqueueFailure(context.Background(), 1)
	metrics.RecordSent(context.Background(), 7)
	require.Equal(t, int64(7), sent)
}

// failBytesSize fails the test when a no-op operation measures the request.
func failBytesSize(t *testing.T) Int64Value {
	return NewInt64Value(func() int64 {
		t.Error("bytes size must not be measured when the operation reports nothing")
		return 0
	})
}

func TestWithObsMetricsNil(t *testing.T) {
	err := WithObsMetrics(nil)(&internal.BaseExporter{})
	require.ErrorContains(t, err, "must not be nil")
}

func TestObsMetricsAdapterForwardsValues(t *testing.T) {
	var values []int64
	recordValue := func(_ context.Context, _ int64, value Int64Value) {
		values = append(values, value.Value())
	}
	metrics := NewObsMetrics(
		WithQueueBatchMetrics(NewQueueBatchMetrics(
			WithQueueMetrics(NewQueueMetrics(
				WithRecordEnqueueSize(recordValue),
				WithRegisterQueueSize(func(value Int64Value) error {
					values = append(values, value.Value())
					return nil
				}),
				WithRegisterQueueCapacity(func(value Int64Value) error {
					values = append(values, value.Value())
					return nil
				}),
			)),
			WithRecordBatchSendSize(recordValue),
		)),
	)
	adapted := adaptObsMetrics(metrics)
	value := queue.NewInt64Value(func() int64 { return 7 })

	adapted.RecordEnqueueSize(context.Background(), 1, value)
	require.NoError(t, adapted.RegisterQueueSize(value))
	require.NoError(t, adapted.RegisterQueueCapacity(value))
	adapted.RecordBatchSendSize(context.Background(), 1, value)
	require.Equal(t, []int64{7, 7, 7, 7}, values)

	noOpAdapted := adaptObsMetrics(NewObsMetrics())
	noOpAdapted.RecordEnqueueSize(context.Background(), 1, nil)
	require.NoError(t, noOpAdapted.RegisterQueueSize(nil))
	require.NoError(t, noOpAdapted.RegisterQueueCapacity(nil))
	noOpAdapted.RecordBatchSendSize(context.Background(), 1, nil)
}
