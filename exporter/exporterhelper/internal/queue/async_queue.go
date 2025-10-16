// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
)

type asyncQueue[T any] struct {
	readableQueue[T]
	numConsumers int
	refCounter   ReferenceCounter[T]
	consumeFunc  ConsumeFunc[T]
	stopWG       sync.WaitGroup
}

func newAsyncQueue[T any](q readableQueue[T], numConsumers int, consumeFunc ConsumeFunc[T], refCounter ReferenceCounter[T]) Queue[T] {
	return &asyncQueue[T]{
		readableQueue: q,
		numConsumers:  numConsumers,
		refCounter:    refCounter,
		consumeFunc:   consumeFunc,
	}
}

// Start ensures that queue and all consumers are started.
func (qc *asyncQueue[T]) Start(ctx context.Context, host component.Host) error {
	if err := qc.readableQueue.Start(ctx, host); err != nil {
		return err
	}
	var startWG sync.WaitGroup
	for i := 0; i < qc.numConsumers; i++ {
		qc.stopWG.Add(1)
		startWG.Add(1)
		go func() { //nolint:contextcheck
			startWG.Done()
			defer qc.stopWG.Done()
			for {
				ctx, req, done, ok := qc.Read(context.Background())
				if !ok {
					return
				}
				qc.consumeFunc(ctx, req, done)
				if qc.refCounter != nil {
					qc.refCounter.Unref(req)
				}
			}
		}()
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
