// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/internal"
	"go.opentelemetry.io/collector/pipeline"
)

// obsQueue is a helper to add observability to a queue.
type obsQueue[T internal.Request] struct {
	exporterqueue.Queue[T]
	tb                *metadata.TelemetryBuilder
	metricAttr        metric.MeasurementOption
	enqueueFailedInst metric.Int64Counter
}

func newObsQueue[T internal.Request](set exporterqueue.Settings, delegate exporterqueue.Queue[T]) (exporterqueue.Queue[T], error) {
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
	}

	switch set.Signal {
	case pipeline.SignalTraces:
		or.enqueueFailedInst = tb.ExporterEnqueueFailedSpans
	case pipeline.SignalMetrics:
		or.enqueueFailedInst = tb.ExporterEnqueueFailedMetricPoints
	case pipeline.SignalLogs:
		or.enqueueFailedInst = tb.ExporterEnqueueFailedLogRecords
	}

	return or, nil
}

func (or *obsQueue[T]) Shutdown(ctx context.Context) error {
	defer or.tb.Shutdown()
	return or.Queue.Shutdown(ctx)
}

func (or *obsQueue[T]) Offer(ctx context.Context, req T) error {
	numItems := req.ItemsCount()
	err := or.Queue.Offer(ctx, req)
	// No metrics recorded for profiles, remove enqueueFailedInst check with nil when profiles metrics available.
	if err != nil && or.enqueueFailedInst != nil {
		or.enqueueFailedInst.Add(ctx, int64(numItems), or.metricAttr)
	}
	return err
}
