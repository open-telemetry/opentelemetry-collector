// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
)

// QueueBatchMetrics reports the metrics produced by a QueueBatch; use NewQueueBatchMetrics.
type QueueBatchMetrics interface {
	queue.QueueMetrics

	// RecordBatchSendSize counts the items and bytes in a batch that is sent.
	RecordBatchSendSize(ctx context.Context, items int64, bytesSize queue.Int64Value)
}

// RecordBatchSendSizeFunc records a batch as it is sent, calling bytesSize only if reported.
type RecordBatchSendSizeFunc func(ctx context.Context, items int64, bytesSize queue.Int64Value)

func (f RecordBatchSendSizeFunc) RecordBatchSendSize(ctx context.Context, items int64, bytesSize queue.Int64Value) {
	if f == nil {
		return
	}
	f(ctx, items, bytesSize)
}

// QueueBatchMetricsOption configures QueueBatchMetrics.
type QueueBatchMetricsOption interface {
	applyQueueBatchMetrics(*queueBatchMetrics)
}

type queueBatchMetricsOptionFunc func(*queueBatchMetrics)

func (f queueBatchMetricsOptionFunc) applyQueueBatchMetrics(metrics *queueBatchMetrics) {
	f(metrics)
}

// WithQueueMetrics configures the queue-level metrics.
func WithQueueMetrics(metrics queue.QueueMetrics) QueueBatchMetricsOption {
	return queueBatchMetricsOptionFunc(func(batchMetrics *queueBatchMetrics) {
		if metrics != nil {
			batchMetrics.QueueMetrics = metrics
		}
	})
}

// WithRecordBatchSendSize configures how batch send sizes are recorded.
func WithRecordBatchSendSize(record RecordBatchSendSizeFunc) QueueBatchMetricsOption {
	return queueBatchMetricsOptionFunc(func(metrics *queueBatchMetrics) {
		metrics.RecordBatchSendSizeFunc = record
	})
}

// NewQueueBatchMetrics returns QueueBatchMetrics whose unspecified operations report nothing.
func NewQueueBatchMetrics(options ...QueueBatchMetricsOption) QueueBatchMetrics {
	metrics := queueBatchMetrics{
		QueueMetrics: queue.NewQueueMetrics(),
	}
	for _, option := range options {
		option.applyQueueBatchMetrics(&metrics)
	}
	return metrics
}

// queueBatchMetrics implements QueueBatchMetrics by extending a QueueMetrics.
type queueBatchMetrics struct {
	queue.QueueMetrics
	RecordBatchSendSizeFunc
}

var _ QueueBatchMetrics = queueBatchMetrics{}
