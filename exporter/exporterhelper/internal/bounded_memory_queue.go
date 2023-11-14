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
type boundedMemoryQueue struct {
	stopWG       sync.WaitGroup
	stopped      *atomic.Bool
	items        chan QueueRequest
	numConsumers int
}

// NewBoundedMemoryQueue constructs the new queue of specified capacity. Capacity cannot be 0.
func NewBoundedMemoryQueue(capacity int, numConsumers int) Queue {
	return &boundedMemoryQueue{
		items:        make(chan QueueRequest, capacity),
		stopped:      &atomic.Bool{},
		numConsumers: numConsumers,
	}
}

// Start starts a given number of goroutines consuming items from the queue
// and passing them into the consumer callback.
func (q *boundedMemoryQueue) Start(_ context.Context, _ component.Host, set QueueSettings) error {
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

// Produce is used by the producer to submit new item to the queue. Returns false in case of queue overflow.
func (q *boundedMemoryQueue) Offer(ctx context.Context, req any) error {
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
func (q *boundedMemoryQueue) Shutdown(context.Context) error {
	q.stopped.Store(true) // disable producer
	close(q.items)
	q.stopWG.Wait()
	return nil
}

// Size returns the current size of the queue
func (q *boundedMemoryQueue) Size() int {
	return len(q.items)
}

func (q *boundedMemoryQueue) Capacity() int {
	return cap(q.items)
}
