// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"
	"errors"
	"sync"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

type asyncQueue[T any] struct {
	readableQueue[T]
	numConsumers      int
	refCounter        ReferenceCounter[T]
	consumeFunc       ConsumeFunc[T]
	waitForCompletion bool
	stopWG            sync.WaitGroup
	inFlight          sync.WaitGroup
}

func newAsyncQueue[T any](q readableQueue[T], numConsumers int, consumeFunc ConsumeFunc[T], refCounter ReferenceCounter[T], waitForCompletion ...bool) Queue[T] {
	wait := len(waitForCompletion) > 0 && waitForCompletion[0]
	return &asyncQueue[T]{
		readableQueue:     q,
		numConsumers:      numConsumers,
		refCounter:        refCounter,
		consumeFunc:       consumeFunc,
		waitForCompletion: wait,
	}
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
				ctx, req, done, ok := qc.Read(context.Background())
				if !ok {
					return
				}
				if qc.waitForCompletion {
					qc.inFlight.Add(1)
					qc.consumeFunc(ctx, req, &asyncDone[T]{
						Done: done, request: req, refCounter: qc.refCounter, inFlight: &qc.inFlight,
					})
					continue
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

func (qc *asyncQueue[T]) LoadCheckpoint(ctx context.Context, key string) ([]byte, bool, error) {
	store, ok := qc.readableQueue.(request.QueueCheckpointStore)
	if !ok {
		return nil, false, nil
	}
	return store.LoadCheckpoint(ctx, key)
}

func (qc *asyncQueue[T]) SaveCheckpoint(ctx context.Context, key string, value []byte) error {
	store, ok := qc.readableQueue.(request.QueueCheckpointStore)
	if !ok {
		return nil
	}
	return store.SaveCheckpoint(ctx, key, value)
}

func (qc *asyncQueue[T]) SaveCheckpointAndItems(ctx context.Context, key string, value []byte, updates []request.QueueItemUpdate) error {
	if store, ok := qc.readableQueue.(request.QueueCheckpointTransaction); ok {
		return store.SaveCheckpointAndItems(ctx, key, value, updates)
	}
	if len(updates) > 0 {
		return errors.New("queue cannot atomically checkpoint item progress")
	}
	return qc.SaveCheckpoint(ctx, key, value)
}

// Shutdown ensures that queue and all consumers are stopped.
func (qc *asyncQueue[T]) Shutdown(ctx context.Context) error {
	err := qc.readableQueue.Shutdown(ctx)
	qc.stopWG.Wait()
	if qc.waitForCompletion {
		qc.inFlight.Wait()
	}
	return err
}

// asyncDone ties the request reference and queue completion to the actual
// downstream completion callback. A consumer may return after starting async
// work, which frees its worker to read another item; the request remains owned
// by the queue until OnDone retires it.
type asyncDone[T any] struct {
	Done
	request    T
	refCounter ReferenceCounter[T]
	inFlight   *sync.WaitGroup
	once       sync.Once
}

func (d *asyncDone[T]) OnDone(err error) {
	d.once.Do(func() {
		defer d.inFlight.Done()
		if d.refCounter != nil {
			defer d.refCounter.Unref(d.request)
		}
		d.Done.OnDone(err)
	})
}

func (qc *asyncQueue[T]) LoadQueueItem(ctx context.Context, token uint64) ([]byte, error) {
	if reader, ok := qc.readableQueue.(request.QueueItemReader); ok {
		return reader.LoadQueueItem(ctx, token)
	}
	return nil, errors.New("queue cannot refresh a durable item")
}
