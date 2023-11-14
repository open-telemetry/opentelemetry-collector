// Copyright The OpenTelemetry Authors
// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/collector/component"
)

// boundedMemoryQueue implements a producer-consumer exchange similar to a ring buffer queue,
// where the queue is bounded and if it fills up due to slow consumers, the new items written by
// the producer are dropped.
type boundedMemoryQueue[T any] struct {
	stopWG       sync.WaitGroup
	stopped      *atomic.Bool
	items        chan QueueRequest[T]
	numConsumers int
}

// NewBoundedMemoryQueue constructs the new queue of specified capacity. Capacity cannot be 0.
func NewBoundedMemoryQueue[T any](capacity int, numConsumers int) Queue[T] {
	return &boundedMemoryQueue[T]{
		items:        make(chan QueueRequest[T], capacity),
		stopped:      &atomic.Bool{},
		numConsumers: numConsumers,
	}
}

// Start starts a given number of goroutines consuming items from the queue
// and passing them into the consumer callback.
func (q *boundedMemoryQueue[T]) Start(_ context.Context, _ component.Host, set QueueSettings[T]) error {
	var startWG sync.WaitGroup
	for i := 0; i < q.numConsumers; i++ {
		q.stopWG.Add(1)
		startWG.Add(1)
		go func() {
			startWG.Done()
			defer q.stopWG.Done()
			for item := range q.items {
				set.Callback(item)
			}
		}()
	}
	startWG.Wait()
	return nil
}

// Offer is used by the producer to submit new item to the queue. Returns false in case of queue overflow.
func (q *boundedMemoryQueue[T]) Offer(ctx context.Context, req T) error {
	if q.stopped.Load() {
		return ErrQueueIsStopped
	}

	select {
	case q.items <- newQueueRequest(ctx, req):
		return nil
	default:
		return ErrQueueIsFull
	}
}

// Shutdown stops accepting items, and stops all consumers. It blocks until all consumers have stopped.
func (q *boundedMemoryQueue[T]) Shutdown(context.Context) error {
	q.stopped.Store(true) // disable producer
	close(q.items)
	q.stopWG.Wait()
	return nil
}

// Size returns the current size of the queue
func (q *boundedMemoryQueue[T]) Size() int {
	return len(q.items)
}

func (q *boundedMemoryQueue[T]) Capacity() int {
	return cap(q.items)
}
