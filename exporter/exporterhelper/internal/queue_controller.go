// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
)

// QueueController is a helper struct that controls queue input and output by limiting queue size and
// number of consumers.
type QueueController[T any] struct {
	mu              sync.Mutex
	capacityLimiter QueueCapacityLimiter[T]
	queue           Queue[T]
	numConsumers    int
	consumeFunc     func(context.Context, T)
	stopWG          sync.WaitGroup
}

func NewQueueController[T any](q Queue[T], capacityLimiter QueueCapacityLimiter[T], numConsumers int,
	consumeFunc func(context.Context, T)) *QueueController[T] {
	return &QueueController[T]{
		queue:           q,
		capacityLimiter: capacityLimiter,
		numConsumers:    numConsumers,
		consumeFunc:     consumeFunc,
		stopWG:          sync.WaitGroup{},
	}
}

// Start ensures that queue and all consumers are started.
func (qc *QueueController[T]) Start(ctx context.Context, host component.Host) error {
	if err := qc.queue.Start(ctx, host); err != nil {
		return err
	}

	var startWG sync.WaitGroup
	for i := 0; i < qc.numConsumers; i++ {
		qc.stopWG.Add(1)
		startWG.Add(1)
		go func() {
			startWG.Done()
			defer qc.stopWG.Done()
			for {
				ok := qc.queue.Consume(func(ctx context.Context, item T) {
					qc.capacityLimiter.Release(item)
					qc.consumeFunc(ctx, item)
				})
				if !ok {
					return
				}
			}
		}()
	}
	startWG.Wait()

	return nil
}

// Shutdown ensures that queue and all consumers are stopped.
func (qc *QueueController[T]) Shutdown(ctx context.Context) error {
	if err := qc.queue.Shutdown(ctx); err != nil {
		return err
	}
	qc.stopWG.Wait()
	return nil
}

func (qc *QueueController[T]) Offer(ctx context.Context, item T) error {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	if !qc.capacityLimiter.Claim(item) {
		return ErrQueueIsFull
	}

	err := qc.queue.Offer(ctx, item)
	if err != nil {
		qc.capacityLimiter.Release(item)
	}
	return err
}

func (qc *QueueController[T]) Size() int {
	return qc.capacityLimiter.Size()
}

func (qc *QueueController[T]) Capacity() int {
	return qc.capacityLimiter.Capacity()
}
