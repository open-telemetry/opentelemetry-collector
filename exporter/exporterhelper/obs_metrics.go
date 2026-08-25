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

// RecordEnqueueFailureFunc records the items dropped because the queue rejected them.
type RecordEnqueueFailureFunc = queue.RecordEnqueueFailureFunc

// RecordEnqueueSizeFunc records a request offered to the queue, calling bytesSize only if reported.
type RecordEnqueueSizeFunc = queue.RecordEnqueueSizeFunc

// RegisterQueueSizeFunc installs an observer for the current queue size.
type RegisterQueueSizeFunc = queue.RegisterQueueSizeFunc

// RegisterQueueCapacityFunc installs an observer for the fixed queue capacity.
type RegisterQueueCapacityFunc = queue.RegisterQueueCapacityFunc

// RecordBatchSendSizeFunc records a batch as it is sent, calling bytesSize only if reported.
type RecordBatchSendSizeFunc = queuebatch.RecordBatchSendSizeFunc

// RecordInFlightFunc records a change in the requests being sent, +1 at start and -1 at end.
type RecordInFlightFunc = internal.RecordInFlightFunc

// RecordSentFunc records the number of items successfully sent.
type RecordSentFunc = internal.RecordSentFunc

// RecordSendFailureFunc records the items that failed to send, with the failure attributes.
type RecordSendFailureFunc = internal.RecordSendFailureFunc

// ShutdownObsMetricsFunc releases the resources backing the instruments.
type ShutdownObsMetricsFunc = internal.ShutdownObsMetricsFunc

// NewQueueMetrics returns a QueueMetrics whose nil operations report nothing.
func NewQueueMetrics(
	recordEnqueueFailure RecordEnqueueFailureFunc,
	recordEnqueueSize RecordEnqueueSizeFunc,
	registerQueueSize RegisterQueueSizeFunc,
	registerQueueCapacity RegisterQueueCapacityFunc,
) QueueMetrics {
	return queue.NewQueueMetrics(recordEnqueueFailure, recordEnqueueSize, registerQueueSize, registerQueueCapacity)
}

// NewQueueBatchMetrics returns a QueueBatchMetrics whose nil arguments report nothing.
func NewQueueBatchMetrics(
	queueMetrics QueueMetrics,
	recordBatchSendSize RecordBatchSendSizeFunc,
) QueueBatchMetrics {
	return queuebatch.NewQueueBatchMetrics(queueMetrics, recordBatchSendSize)
}

// NewObsMetrics returns an ObsMetrics whose nil arguments report nothing.
func NewObsMetrics(
	queueBatchMetrics QueueBatchMetrics,
	recordInFlight RecordInFlightFunc,
	recordSent RecordSentFunc,
	recordSendFailure RecordSendFailureFunc,
	shutdown ShutdownObsMetricsFunc,
) ObsMetrics {
	return internal.NewObsMetrics(queueBatchMetrics, recordInFlight, recordSent, recordSendFailure, shutdown)
}
