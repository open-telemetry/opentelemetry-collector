// Copyright The OpenTelemetry Authors
// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"sync/atomic"

	"go.opentelemetry.io/collector/component"
)

// boundedMemoryQueue implements a producer-consumer exchange similar to a ring buffer queue,
// where the queue is bounded and if it fills up due to slow consumers, the new items written by
// the producer are dropped.
type boundedMemoryQueue[T any] struct {
	component.StartFunc
	stopped *atomic.Bool
	items   chan QueueRequest[T]
}

// NewBoundedMemoryQueue constructs the new queue of specified capacity, and with an optional
// callback for dropped items (e.g. useful to emit metrics).
func NewBoundedMemoryQueue[T any](capacity int) Queue[T] {
	return &boundedMemoryQueue[T]{
		items:   make(chan QueueRequest[T], capacity),
		stopped: &atomic.Bool{},
	}
}

// Offer is used by the producer to submit new item to the queue.
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

// Poll returns a request from the queue once it's available. It returns false if the queue is stopped.
func (q *boundedMemoryQueue[T]) Poll() (QueueRequest[T], bool) {
	item, ok := <-q.items
	return item, ok
}

// Shutdown stops accepting items, and stops all consumers. It blocks until all consumers have stopped.
func (q *boundedMemoryQueue[T]) Shutdown(context.Context) error {
	q.stopped.Store(true) // disable producer
	close(q.items)
	return nil
}

// Size returns the current size of the queue
func (q *boundedMemoryQueue[T]) Size() int {
	return len(q.items)
}

func (q *boundedMemoryQueue[T]) Capacity() int {
	return cap(q.items)
}
