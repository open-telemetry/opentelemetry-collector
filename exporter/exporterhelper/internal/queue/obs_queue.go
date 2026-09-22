// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	queuebatchtelemetry "go.opentelemetry.io/collector/internal/telemetry/queuebatch"
)

const (
	// ExporterKey used to identify exporters in metrics and traces.
	exporterKey = "exporter"

	// DataTypeKey used to identify the data type in the queue size metric.
	dataTypeKey = "data_type"
)

// obsQueue is a helper to add observability to a queue.
type obsQueue[T request.Request] struct {
	Queue[T]
	obsMetrics queuebatchtelemetry.ObsMetrics
	tracer     trace.Tracer
}

func newObsQueue[T request.Request](
	set Settings[T],
	obsMetrics queuebatchtelemetry.ObsMetrics,
	delegate Queue[T],
) (Queue[T], error) {
	if err := obsMetrics.RegisterInt(queuebatchtelemetry.MetricQueueSize, delegate.Size); err != nil {
		return nil, err
	}
	if err := obsMetrics.RegisterInt(queuebatchtelemetry.MetricQueueCapacity, delegate.Capacity); err != nil {
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

	or.obsMetrics.RecordInt(ctx, queuebatchtelemetry.MetricEnqueueSize, int64(numItems))
	if or.obsMetrics.ShouldRecord(ctx, queuebatchtelemetry.MetricEnqueueSizeBytes) {
		or.obsMetrics.RecordInt(ctx, queuebatchtelemetry.MetricEnqueueSizeBytes, int64(req.BytesSize()))
	}

	ctx, span := or.tracer.Start(ctx, "exporter/enqueue")
	err := or.Queue.Offer(ctx, req)
	span.End()

	if err != nil {
		or.obsMetrics.RecordInt(ctx, queuebatchtelemetry.MetricEnqueueFailure, int64(numItems))
	}
	return err
}
