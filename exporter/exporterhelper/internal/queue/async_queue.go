// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

type asyncQueue[T request.Request] struct {
	readableQueue[T]
	numConsumers int
	refCounter   ReferenceCounter[T]
	consumeFunc  ConsumeFunc[T]
	sendAge      metric.Int64Histogram
	metricAttr   metric.MeasurementOption
	stopWG       sync.WaitGroup
}

func newAsyncQueue[T request.Request](q readableQueue[T], set Settings[T], consumeFunc ConsumeFunc[T]) (Queue[T], error) {
	tb, err := metadata.NewTelemetryBuilder(set.Telemetry)
	if err != nil {
		return nil, err
	}

	return &asyncQueue[T]{
		readableQueue: q,
		numConsumers:  set.NumConsumers,
		refCounter:    set.ReferenceCounter,
		consumeFunc:   consumeFunc,
		sendAge:       tb.ExporterQueueBatchSendAge,
		metricAttr:    metric.WithAttributeSet(attribute.NewSet(attribute.String(exporterKey, set.ID.String()), attribute.String(dataTypeKey, set.Signal.String()))),
	}, nil
}

// Start ensures that queue and all consumers are started.
func (qc *asyncQueue[T]) Start(ctx context.Context, host component.Host) error {
	if err := qc.readableQueue.Start(ctx, host); err != nil {
		return err
	}
	var startWG sync.WaitGroup
	for i := 0; i < qc.numConsumers; i++ {
		startWG.Add(1)
		qc.stopWG.Go(func() { //nolint:contextcheck
			startWG.Done()
			for {
				ctx, req, enqueuedAt, done, ok := qc.Read(context.Background())
				if !ok {
					return
				}
				if !enqueuedAt.IsZero() {
					age := time.Since(enqueuedAt).Milliseconds()
					if age < 0 {
						age = 0
					}
					qc.sendAge.Record(ctx, age, qc.metricAttr)
				}
				qc.consumeFunc(ctx, req, done)
				if qc.refCounter != nil {
					qc.refCounter.Unref(req)
				}
			}
		})
	}
	startWG.Wait()

	return nil
}

func (qc *asyncQueue[T]) Offer(ctx context.Context, req T) error {
	span := trace.SpanFromContext(ctx)
	if err := qc.readableQueue.Offer(ctx, req); err != nil {
		span.AddEvent("Failed to enqueue item.")
		return err
	}

	span.AddEvent("Enqueued item.")
	return nil
}

// Shutdown ensures that queue and all consumers are stopped.
func (qc *asyncQueue[T]) Shutdown(ctx context.Context) error {
	err := qc.readableQueue.Shutdown(ctx)
	qc.stopWG.Wait()
	return err
}

func (qc *asyncQueue[T]) OldestTimestamp() time.Time {
	return qc.readableQueue.OldestTimestamp()
}
