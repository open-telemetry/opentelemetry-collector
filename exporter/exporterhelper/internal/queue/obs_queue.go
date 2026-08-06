// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

// QueueObserver observes the current size or capacity of a queue.
type QueueObserver interface {
	Observe() int64
}

// ObserveQueueFunc returns the current size or capacity of a queue.
type ObserveQueueFunc func() int64

func (f ObserveQueueFunc) Observe() int64 {
	if f == nil {
		return 0
	}
	return f()
}

// QueueBatchMetrics reports the metrics produced by a Queue. The queue only
// reports observation events; the caller supplies the instruments and owns
// their lifecycle, because the same instruments may also serve other senders.
type QueueBatchMetrics interface {
	RecordEnqueueFailure(context.Context, int64)
	RecordBatchSendSize(context.Context, int64, int64)
	RegisterQueueSize(QueueObserver) error
	RegisterQueueCapacity(QueueObserver) error
}

type RecordEnqueueFailureFunc func(context.Context, int64)

func (f RecordEnqueueFailureFunc) RecordEnqueueFailure(ctx context.Context, items int64) {
	if f != nil {
		f(ctx, items)
	}
}

type RecordBatchSendSizeFunc func(context.Context, int64, int64)

func (f RecordBatchSendSizeFunc) RecordBatchSendSize(ctx context.Context, items, bytes int64) {
	if f != nil {
		f(ctx, items, bytes)
	}
}

type RegisterQueueSizeFunc func(QueueObserver) error

func (f RegisterQueueSizeFunc) RegisterQueueSize(observe QueueObserver) error {
	if f == nil {
		return nil
	}
	return f(observe)
}

type RegisterQueueCapacityFunc func(QueueObserver) error

func (f RegisterQueueCapacityFunc) RegisterQueueCapacity(observe QueueObserver) error {
	if f == nil {
		return nil
	}
	return f(observe)
}

// FuncQueueBatchMetrics implements QueueBatchMetrics from a set of operations.
// Unset operations are no-ops, so a caller only supplies the events it reports.
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

	if err := obsMetrics.RegisterQueueSize(ObserveQueueFunc(delegate.Size)); err != nil {
		return nil, err
	}

	if err := obsMetrics.RegisterQueueCapacity(ObserveQueueFunc(delegate.Capacity)); err != nil {
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
