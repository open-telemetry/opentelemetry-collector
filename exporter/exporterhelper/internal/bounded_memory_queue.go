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
	size         *atomic.Int32
	stopChan     chan struct{}
	items        chan Request
	capacity     int
	numConsumers int
	stopped      *atomic.Bool
	overflow     *atomic.Bool
}

// NewBoundedMemoryQueue constructs the new queue of specified capacity, and with an optional
// callback for dropped items (e.g. useful to emit metrics).
func NewBoundedMemoryQueue(capacity int, numConsumers int) ProducerConsumerQueue {
	return &boundedMemoryQueue{
		items:        make(chan Request, capacity),
		capacity:     capacity,
		numConsumers: numConsumers,
		size:         &atomic.Int32{},
		stopped:      &atomic.Bool{},
		overflow: func() *atomic.Bool {
			o := &atomic.Bool{}
			if capacity == 0 {
				o.Store(true)
			}
			return o
		}(),
	}
}

// Start starts a given number of goroutines consuming items from the queue
// and passing them into the consumer callback.
func (q *boundedMemoryQueue) Start(_ context.Context, _ component.Host, set QueueSettings) error {
	var startWG sync.WaitGroup

	for i := 0; i < q.numConsumers; i++ {
		startWG.Add(1)
		go func() {
			startWG.Done()
			for item := range q.items {
				set.Callback(item)
				newSize := q.size.Add(-1)
				q.overflow.Store(q.capacity == 0 || newSize >= int32(q.capacity))
				if q.stopChan != nil && newSize == 0 {
					q.stopChan <- struct{}{}
				}
			}
		}()
	}
	startWG.Wait()
	return nil
}

// Produce is used by the producer to submit new item to the queue. Returns false in case of queue overflow.
func (q *boundedMemoryQueue) Produce(item Request) bool {
	if q.stopped.Load() || q.overflow.Load() {
		return false
	}
	newSize := q.size.Add(1)
	q.items <- item
	overflow := q.capacity == 0 || newSize >= int32(q.capacity)
	if overflow {
		q.overflow.Store(true)
	}
	return true
}

// Stop stops all consumers, as well as the length reporter if started,
// and releases the items channel. It blocks until all consumers have stopped.
func (q *boundedMemoryQueue) Stop() {
	q.stopped.Store(true)
	// if we have no consumers, empty the queue, dropping its contents.
	if q.numConsumers == 0 || q.size.Load() == 0 {
		for len(q.items) > 0 {
			<-q.items
			q.size.Add(-1)
		}
		close(q.items)
		return
	}
	s := make(chan struct{})
	q.stopChan = s
	<-s
	close(q.items)
	close(q.stopChan)
}

// Size returns the current size of the queue
func (q *boundedMemoryQueue) Size() int {
	if q.stopped.Load() {
		return 0
	}
	return int(q.size.Load())
}

func (q *boundedMemoryQueue) Capacity() int {
	return q.capacity
}

func (q *boundedMemoryQueue) IsPersistent() bool {
	return false
}
