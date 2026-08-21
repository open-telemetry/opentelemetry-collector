// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
)

// ObsMetrics reports the metrics produced by exporterhelper for one signal; use NewObsMetrics.
type ObsMetrics = internal.ObsMetrics

// QueueMetrics reports the metrics produced by the sending queue; use NewQueueMetrics.
type QueueMetrics = queue.QueueMetrics

// QueueBatchMetrics reports the queue and batcher metrics; use NewQueueBatchMetrics.
type QueueBatchMetrics = queuebatch.QueueBatchMetrics

// Int64Value supplies an int64 value; use NewInt64Value.
type Int64Value = queue.Int64Value

// ValueFunc supplies an int64 value. A nil ValueFunc returns zero.
type ValueFunc = queue.ValueFunc

// RecordEnqueueFailureFunc records the items dropped because the queue rejected them.
// A nil RecordEnqueueFailureFunc reports nothing.
type RecordEnqueueFailureFunc = queue.RecordEnqueueFailureFunc

// RecordEnqueueSizeFunc records a request offered to the queue, calling bytesSize only if reported.
// A nil RecordEnqueueSizeFunc reports nothing.
type RecordEnqueueSizeFunc = queue.RecordEnqueueSizeFunc

// RegisterQueueSizeFunc installs an observer for the current queue size.
// A nil RegisterQueueSizeFunc returns nil without installing an observer.
type RegisterQueueSizeFunc = queue.RegisterQueueSizeFunc

// RegisterQueueCapacityFunc installs an observer for the fixed queue capacity.
// A nil RegisterQueueCapacityFunc returns nil without installing an observer.
type RegisterQueueCapacityFunc = queue.RegisterQueueCapacityFunc

// RecordBatchSendSizeFunc records a batch as it is sent, calling bytesSize only if reported.
// A nil RecordBatchSendSizeFunc reports nothing.
type RecordBatchSendSizeFunc = queuebatch.RecordBatchSendSizeFunc

// RecordInFlightFunc records a change in the requests being sent, +1 at start and -1 at end.
// A nil RecordInFlightFunc reports nothing.
type RecordInFlightFunc = internal.RecordInFlightFunc

// RecordSentFunc records the number of items successfully sent.
// A nil RecordSentFunc reports nothing.
type RecordSentFunc = internal.RecordSentFunc

// RecordSendFailureFunc records the items that failed to send, with the failure attributes.
// A nil RecordSendFailureFunc reports nothing.
type RecordSendFailureFunc = internal.RecordSendFailureFunc

// ShutdownFunc releases the resources backing the instruments.
// A nil ShutdownFunc releases nothing.
type ShutdownFunc = internal.ShutdownFunc

// QueueMetricsOption configures QueueMetrics.
type QueueMetricsOption = queue.QueueMetricsOption

// QueueBatchMetricsOption configures QueueBatchMetrics.
type QueueBatchMetricsOption = queuebatch.QueueBatchMetricsOption

// ObsMetricsOption configures ObsMetrics.
type ObsMetricsOption = internal.ObsMetricsOption

// NewInt64Value returns an Int64Value backed by value.
func NewInt64Value(value ValueFunc) Int64Value {
	return queue.NewInt64Value(value)
}

// WithRecordEnqueueFailure configures how enqueue failures are recorded.
func WithRecordEnqueueFailure(record RecordEnqueueFailureFunc) QueueMetricsOption {
	return queue.WithRecordEnqueueFailure(record)
}

// WithRecordEnqueueSize configures how enqueue sizes are recorded.
func WithRecordEnqueueSize(record RecordEnqueueSizeFunc) QueueMetricsOption {
	return queue.WithRecordEnqueueSize(record)
}

// WithRegisterQueueSize configures how the queue-size observer is registered.
func WithRegisterQueueSize(register RegisterQueueSizeFunc) QueueMetricsOption {
	return queue.WithRegisterQueueSize(register)
}

// WithRegisterQueueCapacity configures how the queue-capacity observer is registered.
func WithRegisterQueueCapacity(register RegisterQueueCapacityFunc) QueueMetricsOption {
	return queue.WithRegisterQueueCapacity(register)
}

// NewQueueMetrics returns QueueMetrics whose unspecified operations report nothing.
func NewQueueMetrics(options ...QueueMetricsOption) QueueMetrics {
	return queue.NewQueueMetrics(options...)
}

// WithQueueMetrics configures the queue-level metrics.
func WithQueueMetrics(metrics QueueMetrics) QueueBatchMetricsOption {
	return queuebatch.WithQueueMetrics(metrics)
}

// WithRecordBatchSendSize configures how batch send sizes are recorded.
func WithRecordBatchSendSize(record RecordBatchSendSizeFunc) QueueBatchMetricsOption {
	return queuebatch.WithRecordBatchSendSize(record)
}

// NewQueueBatchMetrics returns QueueBatchMetrics whose unspecified operations report nothing.
func NewQueueBatchMetrics(options ...QueueBatchMetricsOption) QueueBatchMetrics {
	return queuebatch.NewQueueBatchMetrics(options...)
}

// WithQueueBatchMetrics configures the queue and batch metrics.
func WithQueueBatchMetrics(metrics QueueBatchMetrics) ObsMetricsOption {
	return internal.WithQueueBatchMetrics(metrics)
}

// WithRecordInFlight configures how in-flight request changes are recorded.
func WithRecordInFlight(record RecordInFlightFunc) ObsMetricsOption {
	return internal.WithRecordInFlight(record)
}

// WithRecordSent configures how successfully sent items are recorded.
func WithRecordSent(record RecordSentFunc) ObsMetricsOption {
	return internal.WithRecordSent(record)
}

// WithRecordSendFailure configures how send failures are recorded.
func WithRecordSendFailure(record RecordSendFailureFunc) ObsMetricsOption {
	return internal.WithRecordSendFailure(record)
}

// WithMetricsShutdown configures how metric resources are released.
func WithMetricsShutdown(shutdown ShutdownFunc) ObsMetricsOption {
	return internal.WithMetricsShutdown(shutdown)
}

// NewObsMetrics returns ObsMetrics whose unspecified operations report nothing.
func NewObsMetrics(options ...ObsMetricsOption) ObsMetrics {
	return internal.NewObsMetrics(options...)
}
