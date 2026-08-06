// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

// QueueMetrics reports the metrics produced by a Queue made up
// of two synchronous and two asynchronous instruments.
type QueueMetrics interface {
	// RecordEnqueueFailure counts failures.
	RecordEnqueueFailure(ctx context.Context, items int64)
	// RecordEnqueueItems counts success and bytes.
	RecordEnqueueItems(ctx context.Context, items, bytes int64)
	// RegisterQueueSize is asynchronous.
	RegisterQueueSize(observeSize func() int64) error
	// RegisterQueueCapacity is asynchronous.
	RegisterQueueCapacity(observeCapacity func() int64) error

	sealed()
}

// RecordEnqueueFailureFunc records the number of items dropped because they
// could not be added to the queue.
type RecordEnqueueFailureFunc func(ctx context.Context, items int64)

func (f RecordEnqueueFailureFunc) RecordEnqueueFailure(ctx context.Context, items int64) {
	if f == nil {
		return
	}
	f(ctx, items)
}

// RecordEnqueueItemsFunc records the number of items and bytes in a request
// as it is offered to the queue.
type RecordEnqueueItemsFunc func(ctx context.Context, items, bytes int64)

func (f RecordEnqueueItemsFunc) RecordEnqueueItems(ctx context.Context, items, bytes int64) {
	if f == nil {
		return
	}
	f(ctx, items, bytes)
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

// NewQueueMetrics is sealed.
func NewQueueMetrics(
	ref RecordEnqueueFailureFunc,
	rbss RecordEnqueueItemsFunc,
	rqs RegisterQueueSizeFunc,
	rqc RegisterQueueCapacityFunc,
) QueueMetrics {
	return queueMetrics{
		RecordEnqueueFailureFunc:  ref,
		RecordEnqueueItemsFunc:    rbss,
		RegisterQueueSizeFunc:     rqs,
		RegisterQueueCapacityFunc: rqc,
	}
}

// queueMetrics implements QueueMetrics from a set of operations.
type queueMetrics struct {
	RecordEnqueueFailureFunc
	RecordEnqueueItemsFunc
	RegisterQueueSizeFunc
	RegisterQueueCapacityFunc
}

func (queueMetrics) sealed() {}

var _ QueueMetrics = queueMetrics{}

// obsQueue is a helper to add observability to a queue.
type obsQueue[T request.Request] struct {
	Queue[T]
	queueMetrics QueueMetrics
	tracer       trace.Tracer
}

func newObsQueue[T request.Request](set Settings[T], delegate Queue[T]) (Queue[T], error) {
	var queueMetrics QueueMetrics = queueMetrics{}
	if set.ObsMetrics != nil {
		queueMetrics = set.ObsMetrics
	}

	if err := queueMetrics.RegisterQueueSize(delegate.Size); err != nil {
		return nil, err
	}

	if err := queueMetrics.RegisterQueueCapacity(delegate.Capacity); err != nil {
		return nil, err
	}

	return &obsQueue[T]{
		Queue:      delegate,
		queueMetrics: queueMetrics,
		tracer:     metadata.Tracer(set.Telemetry),
	}, nil
}

func (or *obsQueue[T]) Offer(ctx context.Context, req T) error {
	// Have to read the number of items before sending the request since the request can
	// be modified by the downstream components like the batcher.
	numItems := req.ItemsCount()

	or.queueMetrics.RecordEnqueueItems(ctx, int64(numItems), int64(req.BytesSize()))

	ctx, span := or.tracer.Start(ctx, "exporter/enqueue")
	err := or.Queue.Offer(ctx, req)
	span.End()

	if err != nil {
		or.queueMetrics.RecordEnqueueFailure(ctx, int64(numItems))
	}
	return err
}
