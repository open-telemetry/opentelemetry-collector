// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

const (
	exporterKey = "exporter"
	dataTypeKey = "data_type"
)

// ObsMetrics reports the metrics produced by a Queue.
type ObsMetrics interface {
	RecordEnqueueFailure(context.Context, int64)
	RecordBatchSendSize(context.Context, int64, int64)
	RegisterQueueSize(func() int64) error
	RegisterQueueCapacity(func() int64) error
}

type defaultQueueObsMetrics struct {
	tb                     *metadata.TelemetryBuilder
	metricAttr             metric.MeasurementOption
	asyncAttr              metric.MeasurementOption
	enqueueFailedInst      metric.Int64Counter
	queueBatchSizeInst     metric.Int64Histogram
	queueBatchSizeByteInst metric.Int64Histogram
}

func newDefaultQueueObsMetrics[T request.Request](set Settings[T]) (*defaultQueueObsMetrics, error) {
	tb, err := metadata.NewTelemetryBuilder(set.Telemetry)
	if err != nil {
		return nil, err
	}
	exporterAttr := attribute.String(exporterKey, set.ID.String())
	om := &defaultQueueObsMetrics{
		tb:                     tb,
		metricAttr:             metric.WithAttributeSet(attribute.NewSet(exporterAttr)),
		asyncAttr:              metric.WithAttributeSet(attribute.NewSet(exporterAttr, attribute.String(dataTypeKey, set.Signal.String()))),
		queueBatchSizeInst:     tb.ExporterQueueBatchSendSize,
		queueBatchSizeByteInst: tb.ExporterQueueBatchSendSizeBytes,
	}
	switch set.Signal {
	case pipeline.SignalTraces:
		om.enqueueFailedInst = tb.ExporterEnqueueFailedSpans
	case pipeline.SignalMetrics:
		om.enqueueFailedInst = tb.ExporterEnqueueFailedMetricPoints
	case pipeline.SignalLogs:
		om.enqueueFailedInst = tb.ExporterEnqueueFailedLogRecords
	case xpipeline.SignalProfiles:
		om.enqueueFailedInst = tb.ExporterEnqueueFailedProfileSamples
	}
	return om, nil
}

func (om *defaultQueueObsMetrics) RecordEnqueueFailure(ctx context.Context, items int64) {
	if om.enqueueFailedInst != nil {
		om.enqueueFailedInst.Add(ctx, items, om.metricAttr)
	}
}

func (om *defaultQueueObsMetrics) RecordBatchSendSize(ctx context.Context, items, bytes int64) {
	om.queueBatchSizeInst.Record(ctx, items, om.metricAttr)
	om.queueBatchSizeByteInst.Record(ctx, bytes, om.metricAttr)
}

func (om *defaultQueueObsMetrics) RegisterQueueSize(observe func() int64) error {
	return om.tb.RegisterExporterQueueSizeCallback(func(_ context.Context, o metric.Int64Observer) error {
		o.Observe(observe(), om.asyncAttr)
		return nil
	})
}

func (om *defaultQueueObsMetrics) RegisterQueueCapacity(observe func() int64) error {
	return om.tb.RegisterExporterQueueCapacityCallback(func(_ context.Context, o metric.Int64Observer) error {
		o.Observe(observe(), om.asyncAttr)
		return nil
	})
}

// obsQueue is a helper to add observability to a queue.
type obsQueue[T request.Request] struct {
	Queue[T]
	obsMetrics ObsMetrics
	tb         *metadata.TelemetryBuilder
	tracer     trace.Tracer
}

func newObsQueue[T request.Request](set Settings[T], delegate Queue[T]) (Queue[T], error) {
	obsMetrics := set.ObsMetrics
	var tb *metadata.TelemetryBuilder
	if obsMetrics == nil {
		defaultMetrics, err := newDefaultQueueObsMetrics(set)
		if err != nil {
			return nil, err
		}
		obsMetrics = defaultMetrics
		tb = defaultMetrics.tb
	}
	if err := obsMetrics.RegisterQueueSize(delegate.Size); err != nil {
		return nil, err
	}

	if err := obsMetrics.RegisterQueueCapacity(delegate.Capacity); err != nil {
		return nil, err
	}

	tracer := metadata.Tracer(set.Telemetry)

	or := &obsQueue[T]{
		Queue:      delegate,
		obsMetrics: obsMetrics,
		tb:         tb,
		tracer:     tracer,
	}

	return or, nil
}

func (or *obsQueue[T]) Shutdown(ctx context.Context) error {
	if or.tb != nil {
		defer or.tb.Shutdown()
	}
	return or.Queue.Shutdown(ctx)
}

func (or *obsQueue[T]) Offer(ctx context.Context, req T) error {
	// Have to read the number of items before sending the request since the request can
	// be modified by the downstream components like the batcher.
	numItems := req.ItemsCount()

	or.obsMetrics.RecordBatchSendSize(ctx, int64(numItems), int64(req.BytesSize()))

	ctx, span := or.tracer.Start(ctx, "exporter/enqueue")
	err := or.Queue.Offer(ctx, req)
	span.End()

	// No metrics recorded for profiles, remove enqueueFailedInst check with nil when profiles metrics available.
	if err != nil {
		or.obsMetrics.RecordEnqueueFailure(ctx, int64(numItems))
	}
	return err
}
