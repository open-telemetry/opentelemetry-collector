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
	RecordEnqueueFailure(context.Context, int64)
	RecordBatchSendSize(context.Context, int64, int64)
	RegisterQueueSize(observeSize func() int64) error
	RegisterQueueCapacity(observeCapacity func() int64) error
}

// nopMetrics is used when no QueueBatchMetrics is supplied.
type nopMetrics struct{}

func (nopMetrics) RecordEnqueueFailure(context.Context, int64)       {}
func (nopMetrics) RecordBatchSendSize(context.Context, int64, int64) {}
func (nopMetrics) RegisterQueueSize(func() int64) error              { return nil }
func (nopMetrics) RegisterQueueCapacity(func() int64) error          { return nil }

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
	obsMetrics := QueueBatchMetrics(nopMetrics{})
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
