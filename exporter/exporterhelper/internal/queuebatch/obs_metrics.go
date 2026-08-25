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
	RecordBatchSendSize(ctx context.Context, items int64, bytesSize func() int64)
}

// RecordBatchSendSizeFunc records a batch as it is sent, calling bytesSize only if reported.
type RecordBatchSendSizeFunc func(ctx context.Context, items int64, bytesSize func() int64)

func (f RecordBatchSendSizeFunc) RecordBatchSendSize(ctx context.Context, items int64, bytesSize func() int64) {
	if f == nil {
		return
	}
	f(ctx, items, bytesSize)
}

// NewQueueBatchMetrics returns a QueueBatchMetrics whose nil arguments report nothing.
func NewQueueBatchMetrics(
	queueMetrics queue.QueueMetrics,
	recordBatchSendSize RecordBatchSendSizeFunc,
) QueueBatchMetrics {
	if queueMetrics == nil {
		queueMetrics = queue.NewQueueMetrics(nil, nil, nil, nil)
	}
	return queueBatchMetrics{
		QueueMetrics:            queueMetrics,
		RecordBatchSendSizeFunc: recordBatchSendSize,
	}
}

// queueBatchMetrics implements QueueBatchMetrics by extending a QueueMetrics.
type queueBatchMetrics struct {
	queue.QueueMetrics
	RecordBatchSendSizeFunc
}

var _ QueueBatchMetrics = queueBatchMetrics{}
