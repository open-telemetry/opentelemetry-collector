// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

// QueueMetrics reports the metrics produced by a Queue; use NewQueueMetrics.
type QueueMetrics interface {
	// RecordEnqueueFailure counts items the queue rejected.
	RecordEnqueueFailure(ctx context.Context, items int64)
	// RecordEnqueueSize counts the items and bytes offered to the queue.
	RecordEnqueueSize(ctx context.Context, items int64, bytesSize func() int64)
	// RegisterQueueSize installs an observer for the current queue size.
	RegisterQueueSize(observeSize func() int64) error
	// RegisterQueueCapacity installs an observer for the fixed queue capacity.
	RegisterQueueCapacity(observeCapacity func() int64) error

	private()
}

// RecordEnqueueFailureFunc records the items dropped because the queue rejected them.
type RecordEnqueueFailureFunc func(ctx context.Context, items int64)

func (f RecordEnqueueFailureFunc) RecordEnqueueFailure(ctx context.Context, items int64) {
	if f == nil {
		return
	}
	f(ctx, items)
}

// RecordEnqueueSizeFunc records a request offered to the queue, calling bytesSize only if reported.
type RecordEnqueueSizeFunc func(ctx context.Context, items int64, bytesSize func() int64)

func (f RecordEnqueueSizeFunc) RecordEnqueueSize(ctx context.Context, items int64, bytesSize func() int64) {
	if f == nil {
		return
	}
	f(ctx, items, bytesSize)
}

// RegisterQueueSizeFunc installs an observer for the current queue size.
type RegisterQueueSizeFunc func(observeSize func() int64) error

func (f RegisterQueueSizeFunc) RegisterQueueSize(observeSize func() int64) error {
	if f == nil {
		return nil
	}
	return f(observeSize)
}

// RegisterQueueCapacityFunc installs an observer for the fixed queue capacity.
type RegisterQueueCapacityFunc func(observeCapacity func() int64) error

func (f RegisterQueueCapacityFunc) RegisterQueueCapacity(observeCapacity func() int64) error {
	if f == nil {
		return nil
	}
	return f(observeCapacity)
}

// NewQueueMetrics returns a QueueMetrics whose nil operations report nothing.
func NewQueueMetrics(
	recordEnqueueFailure RecordEnqueueFailureFunc,
	recordEnqueueSize RecordEnqueueSizeFunc,
	registerQueueSize RegisterQueueSizeFunc,
	registerQueueCapacity RegisterQueueCapacityFunc,
) QueueMetrics {
	return queueMetrics{
		RecordEnqueueFailureFunc:  recordEnqueueFailure,
		RecordEnqueueSizeFunc:     recordEnqueueSize,
		RegisterQueueSizeFunc:     registerQueueSize,
		RegisterQueueCapacityFunc: registerQueueCapacity,
	}
}

// queueMetrics implements QueueMetrics from a set of operations.
type queueMetrics struct {
	RecordEnqueueFailureFunc
	RecordEnqueueSizeFunc
	RegisterQueueSizeFunc
	RegisterQueueCapacityFunc
}

func (queueMetrics) private() {}

var _ QueueMetrics = queueMetrics{}

// obsQueue is a helper to add observability to a queue.
type obsQueue[T request.Request] struct {
	Queue[T]
	queueMetrics QueueMetrics
	tracer       trace.Tracer
}

func newObsQueue[T request.Request](set Settings[T], delegate Queue[T]) (Queue[T], error) {
	qm := set.QueueMetrics
	if qm == nil {
		qm = NewQueueMetrics(nil, nil, nil, nil)
	}

	if err := qm.RegisterQueueSize(delegate.Size); err != nil {
		return nil, err
	}

	if err := qm.RegisterQueueCapacity(delegate.Capacity); err != nil {
		return nil, err
	}

	return &obsQueue[T]{
		Queue:        delegate,
		queueMetrics: qm,
		tracer:       metadata.Tracer(set.Telemetry),
	}, nil
}

func (or *obsQueue[T]) Offer(ctx context.Context, req T) error {
	// Have to read the number of items before sending the request since the request can
	// be modified by the downstream components like the batcher.
	numItems := req.ItemsCount()

	or.queueMetrics.RecordEnqueueSize(ctx, int64(numItems), func() int64 { return int64(req.BytesSize()) })

	ctx, span := or.tracer.Start(ctx, "exporter/enqueue")
	err := or.Queue.Offer(ctx, req)
	span.End()

	if err != nil {
		or.queueMetrics.RecordEnqueueFailure(ctx, int64(numItems))
	}
	return err
}
