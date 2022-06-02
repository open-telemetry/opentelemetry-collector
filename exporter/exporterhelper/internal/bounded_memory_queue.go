// Copyright The OpenTelemetry Authors
// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"sync"

	"go.uber.org/atomic"
)

// boundedMemoryQueue implements a producer-consumer exchange similar to a ring buffer queue,
// where the queue is bounded and if it fills up due to slow consumers, the new items written by
// the producer force the earliest items to be dropped. The implementation is actually based on
// channels, with a special Reaper goroutine that wakes up when the queue is full and consumers
// the items from the top of the queue until its size drops back to maxSize
type boundedMemoryQueue struct {
	stopWG        sync.WaitGroup
	size          *atomic.Uint32
	stopped       *atomic.Bool
	items         chan interface{}
	onDroppedItem func(item interface{})
	factory       func() consumer
	capacity      uint32
}

// NewBoundedMemoryQueue constructs the new queue of specified capacity, and with an optional
// callback for dropped items (e.g. useful to emit metrics).
func NewBoundedMemoryQueue(capacity int, onDroppedItem func(item interface{})) ProducerConsumerQueue {
	return &boundedMemoryQueue{
		onDroppedItem: onDroppedItem,
		items:         make(chan interface{}, capacity),
		stopped:       atomic.NewBool(false),
		size:          atomic.NewUint32(0),
		capacity:      uint32(capacity),
	}
}

// StartConsumers starts a given number of goroutines consuming items from the queue
// and passing them into the consumer callback.
func (q *boundedMemoryQueue) StartConsumers(numWorkers int, callback func(item interface{})) {
	factory := func() consumer {
		return consumerFunc(callback)
	}
	q.factory = factory
	var startWG sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		q.stopWG.Add(1)
		startWG.Add(1)
		go func() {
			startWG.Done()
			defer q.stopWG.Done()
			itemConsumer := q.factory()
			for item := range q.items {
				q.size.Sub(1)
				itemConsumer.consume(item)
			}
		}()
	}
	startWG.Wait()
}

// consumerFunc is an adapter to allow the use of
// a consume function callback as a consumer.
type consumerFunc func(item interface{})

// Consume calls c(item)
func (c consumerFunc) consume(item interface{}) {
	c(item)
}

// Produce is used by the producer to submit new item to the queue. Returns false in case of queue overflow.
func (q *boundedMemoryQueue) Produce(item interface{}) bool {
	if q.stopped.Load() {
		q.onDroppedItem(item)
		return false
	}

	// we might have two concurrent backing queues at the moment
	// their combined size is stored in q.size, and their combined capacity
	// should match the capacity of the new queue
	if q.size.Load() >= q.capacity {
		// note that all items will be dropped if the capacity is 0
		q.onDroppedItem(item)
		return false
	}

	q.size.Add(1)
	select {
	case q.items <- item:
		return true
	default:
		// should not happen, as overflows should have been captured earlier
		q.size.Sub(1)
		if q.onDroppedItem != nil {
			q.onDroppedItem(item)
		}
		return false
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
