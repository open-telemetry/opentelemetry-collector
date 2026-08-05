// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

const (
	exporterKey = "exporter"
	dataTypeKey = "data_type"
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

// Metrics reports the metrics produced by a Queue. The caller owns its
// lifecycle because the same metrics instance may also serve other senders.
type Metrics interface {
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

type exporterQueueMetrics struct {
	RecordEnqueueFailureFunc
	RecordBatchSendSizeFunc
	RegisterQueueSizeFunc
	RegisterQueueCapacityFunc
}

// NewExporterMetrics returns Metrics reporting the exporter-oriented queue
// instruments of tb. The caller owns tb and is responsible for shutting it
// down, because the same builder may also back non-queue instruments.
func NewExporterMetrics(tb *metadata.TelemetryBuilder, id component.ID, signal pipeline.Signal) Metrics {
	exporterAttr := attribute.String(exporterKey, id.String())
	metricAttr := metric.WithAttributeSet(attribute.NewSet(exporterAttr))
	asyncAttr := metric.WithAttributeSet(attribute.NewSet(exporterAttr, attribute.String(dataTypeKey, signal.String())))
	om := &exporterQueueMetrics{
		RecordBatchSendSizeFunc: func(ctx context.Context, items, bytes int64) {
			tb.ExporterQueueBatchSendSize.Record(ctx, items, metricAttr)
			tb.ExporterQueueBatchSendSizeBytes.Record(ctx, bytes, metricAttr)
		},
		RegisterQueueSizeFunc: func(observe QueueObserver) error {
			return tb.RegisterExporterQueueSizeCallback(func(_ context.Context, o metric.Int64Observer) error {
				o.Observe(observe.Observe(), asyncAttr)
				return nil
			})
		},
		RegisterQueueCapacityFunc: func(observe QueueObserver) error {
			return tb.RegisterExporterQueueCapacityCallback(func(_ context.Context, o metric.Int64Observer) error {
				o.Observe(observe.Observe(), asyncAttr)
				return nil
			})
		},
	}
	var enqueueFailedInst metric.Int64Counter
	switch signal {
	case pipeline.SignalTraces:
		enqueueFailedInst = tb.ExporterEnqueueFailedSpans
	case pipeline.SignalMetrics:
		enqueueFailedInst = tb.ExporterEnqueueFailedMetricPoints
	case pipeline.SignalLogs:
		enqueueFailedInst = tb.ExporterEnqueueFailedLogRecords
	case xpipeline.SignalProfiles:
		enqueueFailedInst = tb.ExporterEnqueueFailedProfileSamples
	}
	// enqueueFailedInst is nil only when the signal is not one of the four
	// known signals, in which case enqueue failures go unreported.
	if enqueueFailedInst != nil {
		om.RecordEnqueueFailureFunc = func(ctx context.Context, items int64) {
			enqueueFailedInst.Add(ctx, items, metricAttr)
		}
	}
	return om
}

// obsQueue is a helper to add observability to a queue.
type obsQueue[T request.Request] struct {
	Queue[T]
	obsMetrics Metrics
	tb         *metadata.TelemetryBuilder
	tracer     trace.Tracer
}

func newObsQueue[T request.Request](set Settings[T], delegate Queue[T]) (Queue[T], error) {
	obsMetrics := set.ObsMetrics
	// tb is non-nil only when this queue owns the telemetry builder. Callers
	// reaching the queue through exporterhelper always supply ObsMetrics, so
	// this branch covers direct users of the queue packages.
	var tb *metadata.TelemetryBuilder
	if obsMetrics == nil {
		var err error
		tb, err = metadata.NewTelemetryBuilder(set.Telemetry)
		if err != nil {
			return nil, err
		}
		obsMetrics = NewExporterMetrics(tb, set.ID, set.Signal)
	}
	if err := obsMetrics.RegisterQueueSize(ObserveQueueFunc(delegate.Size)); err != nil {
		shutdownOwnedBuilder(tb)
		return nil, err
	}

	if err := obsMetrics.RegisterQueueCapacity(ObserveQueueFunc(delegate.Capacity)); err != nil {
		shutdownOwnedBuilder(tb)
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
	defer shutdownOwnedBuilder(or.tb)
	return or.Queue.Shutdown(ctx)
}

// shutdownOwnedBuilder releases tb when this queue owns it. tb is nil when the
// caller supplied Metrics, in which case the caller owns the builder.
func shutdownOwnedBuilder(tb *metadata.TelemetryBuilder) {
	if tb != nil {
		tb.Shutdown()
	}
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
