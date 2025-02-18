// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/pipeline"
)

// obsQueue is a helper to add observability to a queue.
type obsQueue[T request.Request] struct {
	exporterqueue.Queue[T]
	tb         *metadata.TelemetryBuilder
	metricAttr metric.MeasurementOption
	signal     pipeline.Signal
}

func newObsQueue[T request.Request](set exporterqueue.Settings, delegate exporterqueue.Queue[T]) (exporterqueue.Queue[T], error) {
	tb, err := metadata.NewTelemetryBuilder(set.ExporterSettings.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	exporterAttr := attribute.String(ExporterKey, set.ExporterSettings.ID.String())
	asyncAttr := metric.WithAttributeSet(attribute.NewSet(exporterAttr, attribute.String(DataTypeKey, set.Signal.String())))
	err = tb.RegisterExporterQueueSizeCallback(func(_ context.Context, o metric.Int64Observer) error {
		o.Observe(delegate.Size(), asyncAttr)
		return nil
	})
	if err != nil {
		return nil, err
	}

	err = tb.RegisterExporterQueueCapacityCallback(func(_ context.Context, o metric.Int64Observer) error {
		o.Observe(delegate.Capacity(), asyncAttr)
		return nil
	})
	if err != nil {
		return nil, err
	}

	or := &obsQueue[T]{
		Queue:      delegate,
		tb:         tb,
		metricAttr: metric.WithAttributeSet(attribute.NewSet(exporterAttr)),
		signal:     set.Signal,
	}

	return or, nil
}

func (or *obsQueue[T]) Shutdown(ctx context.Context) error {
	defer or.tb.Shutdown()
	return or.Queue.Shutdown(ctx)
}

func (or *obsQueue[T]) Offer(ctx context.Context, req T) error {
	// Have to read the number of items before sending the request since the request can
	// be modified by the downstream components like the batcher.
	numItems := req.ItemsCount()
	err := or.Queue.Offer(ctx, req)
	// No metrics recorded for profiles, remove enqueueFailedInst check with nil when profiles metrics available.
	if err != nil {
		switch or.signal {
		case pipeline.SignalTraces:
			or.tb.RecordExporterEnqueueFailedSpans(ctx, int64(numItems), or.metricAttr)
		case pipeline.SignalMetrics:
			or.tb.RecordExporterEnqueueFailedMetricPoints(ctx, int64(numItems), or.metricAttr)
		case pipeline.SignalLogs:
			or.tb.RecordExporterEnqueueFailedLogRecords(ctx, int64(numItems), or.metricAttr)
		}
	}
	return err
}
