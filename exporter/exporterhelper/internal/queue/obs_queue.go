// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/obsmetrics"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

// obsQueue is a helper to add observability to a queue.
type obsQueue[T request.Request] struct {
	Queue[T]
	obsMetrics obsmetrics.ObsMetrics
	tracer     trace.Tracer
}

func newObsQueue[T request.Request](set Settings[T], delegate Queue[T]) (Queue[T], error) {
	if set.ObsMetrics == nil {
		return nil, errors.New("ObsMetrics must not be nil")
	}

	if err := set.ObsMetrics.RegisterQueueSize(delegate.Size); err != nil {
		return nil, err
	}

	if err := set.ObsMetrics.RegisterQueueCapacity(delegate.Capacity); err != nil {
		return nil, err
	}

	return &obsQueue[T]{
		Queue:      delegate,
		obsMetrics: set.ObsMetrics,
		tracer:     metadata.Tracer(set.Telemetry),
	}, nil
}

func (or *obsQueue[T]) Offer(ctx context.Context, req T) error {
	// Have to read the number of items before sending the request since the request can
	// be modified by the downstream components like the batcher.
	numItems := req.ItemsCount()

	or.obsMetrics.RecordEnqueueSize(ctx, int64(numItems), func() int64 {
		return int64(req.BytesSize())
	})

	ctx, span := or.tracer.Start(ctx, "exporter/enqueue")
	err := or.Queue.Offer(ctx, req)
	span.End()

	if err != nil {
		or.obsMetrics.RecordEnqueueFailure(ctx, int64(numItems))
	}
	return err
}
