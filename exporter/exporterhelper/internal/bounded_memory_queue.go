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
	size         *atomic.Uint32
	stopped      *atomic.Bool
	items        chan Request
	capacity     uint32
	numConsumers int
	errCh        chan error
}

// NewBoundedMemoryQueue constructs the new queue of specified capacity, and with an optional
// callback for dropped items (e.g. useful to emit metrics).
func NewBoundedMemoryQueue(capacity int, numConsumers int) ProducerConsumerQueue {
	return &boundedMemoryQueue{
		items:        make(chan Request, capacity),
		stopped:      &atomic.Bool{},
		size:         &atomic.Uint32{},
		capacity:     uint32(capacity),
		numConsumers: numConsumers,
		errCh:        make(chan error, numConsumers),
	}
}

// StartConsumers starts a given number of goroutines consuming items from the queue
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
				q.size.Add(^uint32(0))
				set.Callback(item)
			}
		}()
	}
	startWG.Wait()
	return nil
}

// Produce is used by the producer to submit new item to the queue. Returns false in case of queue overflow.
func (q *boundedMemoryQueue) Produce(item Request) bool {
	if q.stopped.Load() {
		return false
	}

	// we might have two concurrent backing queues at the moment
	// their combined size is stored in q.size, and their combined capacity
	// should match the capacity of the new queue
	if q.size.Load() >= q.capacity {
		return false
	}

	q.size.Add(1)
	select {
	case q.items <- item:
		return true
	default:
		// should not happen, as overflows should have been captured earlier
		q.size.Add(^uint32(0))
		return false
	}
}

// // Same as Produce but waits for response before queuing the next item
func (q *boundedMemoryQueue) ProduceAndWait(item Request, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	if q.stopped.Load() {
		return fmt.Errorf("queue stopped")
	}

	if q.size.Load() < q.capacity {
		q.size.Add(1)
	}

	select {
	case <-timer.C:
		return fmt.Errorf("failed to add item to queue, timeout exceeded")
	case q.items <- item:
		return nil
	}
}

// Stop stops all consumers, as well as the length reporter if started,
// and releases the items channel. It blocks until all consumers have stopped.
func (q *boundedMemoryQueue) Stop() {
	q.stopped.Store(true) // disable producer
	close(q.items)
	q.stopWG.Wait()
}

// Size returns the current size of the queue
func (q *boundedMemoryQueue) Size() int {
	return int(q.size.Load())
}

func (q *boundedMemoryQueue) Capacity() int {
	return int(q.capacity)
}

func (q *boundedMemoryQueue) IsPersistent() bool {
	return false
}
