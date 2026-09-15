// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	queuebatchtelemetry "go.opentelemetry.io/collector/internal/telemetry/queuebatch"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

const (
	exporterKey = "exporter"
	dataTypeKey = "data_type"
)

// obsQueue is a helper to add observability to a queue.
type obsQueue[T request.Request] struct {
	Queue[T]
	obsMetrics *queuebatchtelemetry.ObsMetrics
	shutdown   queuebatchtelemetry.ShutdownFunc
	tracer     trace.Tracer
}

func newObsQueue[T request.Request](set Settings[T], delegate Queue[T]) (Queue[T], error) {
	obsMetrics := set.ObsMetrics
	var shutdown queuebatchtelemetry.ShutdownFunc
	if obsMetrics == nil {
		var err error
		obsMetrics, err = newExporterObsMetrics(set)
		if err != nil {
			return nil, err
		}
		shutdown = obsMetrics.ShutdownFunc
	}

	if err := obsMetrics.RegisterQueue(delegate.Size, delegate.Capacity); err != nil {
		shutdown.Shutdown()
		return nil, err
	}

	return &obsQueue[T]{
		Queue:      delegate,
		obsMetrics: obsMetrics,
		shutdown:   shutdown,
		tracer:     metadata.Tracer(set.Telemetry),
	}, nil
}

func newExporterObsMetrics[T request.Request](set Settings[T]) (*queuebatchtelemetry.ObsMetrics, error) {
	tb, err := metadata.NewTelemetryBuilder(set.Telemetry)
	if err != nil {
		return nil, err
	}

	exporterAttr := attribute.String(exporterKey, set.ID.String())
	metricAttr := metric.WithAttributeSet(attribute.NewSet(exporterAttr))
	queueAttr := metric.WithAttributeSet(attribute.NewSet(
		exporterAttr,
		attribute.String(dataTypeKey, set.Signal.String()),
	))
	shutdown := sync.OnceFunc(tb.Shutdown)

	var enqueueFailedInst metric.Int64Counter
	switch set.Signal {
	case pipeline.SignalTraces:
		enqueueFailedInst = tb.ExporterEnqueueFailedSpans
	case pipeline.SignalMetrics:
		enqueueFailedInst = tb.ExporterEnqueueFailedMetricPoints
	case pipeline.SignalLogs:
		enqueueFailedInst = tb.ExporterEnqueueFailedLogRecords
	case xpipeline.SignalProfiles:
		enqueueFailedInst = tb.ExporterEnqueueFailedProfileSamples
	}

	return &queuebatchtelemetry.ObsMetrics{
		EnqueueFailureFunc: func(ctx context.Context, items int64) {
			if enqueueFailedInst != nil {
				enqueueFailedInst.Add(ctx, items, metricAttr)
			}
		},
		EnqueueSizeFunc: func(ctx context.Context, items int64, bytesSize func() int64) {
			tb.ExporterEnqueueSize.Record(ctx, items, metricAttr)
			if tb.ExporterEnqueueSizeBytes.Enabled(ctx) {
				tb.ExporterEnqueueSizeBytes.Record(ctx, bytesSize(), metricAttr)
			}
		},
		RegisterQueueFunc: func(size, capacity func() int64) error {
			if err := tb.RegisterExporterQueueSizeCallback(func(_ context.Context, o metric.Int64Observer) error {
				o.Observe(size(), queueAttr)
				return nil
			}); err != nil {
				return err
			}
			if err := tb.RegisterExporterQueueCapacityCallback(func(_ context.Context, o metric.Int64Observer) error {
				o.Observe(capacity(), queueAttr)
				return nil
			}); err != nil {
				shutdown()
				return err
			}
			return nil
		},
		ShutdownFunc: shutdown,
	}, nil
}

func (or *obsQueue[T]) Shutdown(ctx context.Context) error {
	defer or.shutdown.Shutdown()
	return or.Queue.Shutdown(ctx)
}

func (or *obsQueue[T]) Offer(ctx context.Context, req T) error {
	// Have to read the number of items before sending the request since the request can
	// be modified by the downstream components like the batcher.
	numItems := req.ItemsCount()

	or.obsMetrics.EnqueueSize(ctx, int64(numItems), func() int64 {
		return int64(req.BytesSize())
	})

	ctx, span := or.tracer.Start(ctx, "exporter/enqueue")
	err := or.Queue.Offer(ctx, req)
	span.End()

	if err != nil {
		or.obsMetrics.EnqueueFailure(ctx, int64(numItems))
	}
	return err
}
