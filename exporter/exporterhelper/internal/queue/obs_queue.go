// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

// QueueBatchMetrics reports the metrics produced by a Queue. The queue only
// reports observation events; the caller supplies the instruments and owns
// their lifecycle, because the same instruments may also serve other senders.
type QueueBatchMetrics interface {
	RecordEnqueueFailure(ctx context.Context, items int64)
	RecordBatchSendSize(ctx context.Context, items, bytes int64)
	RegisterQueueSize(observeSize func() int64) error
	RegisterQueueCapacity(observeCapacity func() int64) error
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

// RecordBatchSendSizeFunc records the number of items and bytes in a request
// as it is offered to the queue.
type RecordBatchSendSizeFunc func(ctx context.Context, items, bytes int64)

func (f RecordBatchSendSizeFunc) RecordBatchSendSize(ctx context.Context, items, bytes int64) {
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

// FuncQueueBatchMetrics implements QueueBatchMetrics from a set of operations.
// The zero value reports nothing, so a caller only sets the events it reports.
type FuncQueueBatchMetrics struct {
	RecordEnqueueFailureFunc
	RecordBatchSendSizeFunc
	RegisterQueueSizeFunc
	RegisterQueueCapacityFunc
}

// obsQueue is a helper to add observability to a queue.
type obsQueue[T request.Request] struct {
	Queue[T]
	obsMetrics QueueBatchMetrics
	tracer     trace.Tracer
}

func newObsQueue[T request.Request](set Settings[T], delegate Queue[T]) (Queue[T], error) {
	// Settings.ObsMetrics is optional: a queue created without it reports no
	// metrics. The caller owns the instruments and their lifecycle, so the
	// queue never creates or shuts down telemetry of its own.
	obsMetrics := QueueBatchMetrics(FuncQueueBatchMetrics{})
	if set.ObsMetrics != nil {
		obsMetrics = set.ObsMetrics
	}

	if err := obsMetrics.RegisterQueueSize(delegate.Size); err != nil {
		return nil, err
	}

	if err := obsMetrics.RegisterQueueCapacity(delegate.Capacity); err != nil {
		return nil, err
	}

	return &obsQueue[T]{
		Queue:      delegate,
		obsMetrics: obsMetrics,
		tracer:     metadata.Tracer(set.Telemetry),
	}, nil
}

func (or *obsQueue[T]) Offer(ctx context.Context, req T) error {
	// Have to read the number of items before sending the request since the request can
	// be modified by the downstream components like the batcher.
	numItems := req.ItemsCount()

	or.obsMetrics.RecordBatchSendSize(ctx, int64(numItems), int64(req.BytesSize()))

	ctx, span := or.tracer.Start(ctx, "exporter/enqueue")
	err := or.Queue.Offer(ctx, req)
	span.End()

	if err != nil {
		or.obsMetrics.RecordEnqueueFailure(ctx, int64(numItems))
	}
	return err
}
