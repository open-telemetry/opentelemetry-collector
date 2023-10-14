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
	size         int
	eventChan    chan any
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
		eventChan:    make(chan any),
		capacity:     capacity,
		numConsumers: numConsumers,
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
	startWG.Add(1)
	go func() {
		startWG.Done()
		q.eventLoop()
	}()

	for i := 0; i < q.numConsumers; i++ {
		startWG.Add(1)
		go func() {
			startWG.Done()
			for item := range q.items {
				set.Callback(item)
				q.eventChan <- true
			}
		}()
	}
	startWG.Wait()
	return nil
}

func (q *boundedMemoryQueue) eventLoop() {
	overflow := q.capacity == 0
	var exitChan chan struct{}
	for {
		e := <-q.eventChan
		if req, ok := e.(Request); ok {
			if overflow {
				continue
			}
			q.size++
			overflow = q.capacity == 0 || q.size >= q.capacity
			if overflow {
				q.overflow.Store(true)
			}
			q.items <- req
			continue
		}
		if done, ok := e.(bool); ok && done {
			q.size--
			if exitChan != nil && q.size == 0 {
				break
			}
			previousOverflow := overflow
			overflow = q.capacity == 0 || q.size >= q.capacity
			if overflow != previousOverflow {
				q.overflow.Store(false)
			}
			continue
		}
		if sizeChan, ok := e.(chan int); ok {
			sizeChan <- q.size
			continue
		}

		if stopChan, ok := e.(chan struct{}); ok {
			// mark the event loop stopped.
			q.stopped.Store(true)
			// if we have no consumers, empty the queue, dropping its contents.
			if q.numConsumers == 0 {
				for len(q.items) > 0 {
					<-q.items
					q.size--
				}
			}
			exitChan = stopChan
			if q.size > 0 {
				// we have a stop signal, but there are still elements in the queue.
				continue
			}
			break
		}
	}
	close(q.items)
	close(exitChan)
	close(q.eventChan)
}

// Produce is used by the producer to submit new item to the queue. Returns false in case of queue overflow.
func (q *boundedMemoryQueue) Produce(item Request) bool {
	if q.stopped.Load() || q.overflow.Load() {
		return false
	}
	q.eventChan <- item
	return true
}

// Stop stops all consumers, as well as the length reporter if started,
// and releases the items channel. It blocks until all consumers have stopped.
func (q *boundedMemoryQueue) Stop() {
	stopChan := make(chan struct{})
	q.eventChan <- stopChan
	<-stopChan
}

// Size returns the current size of the queue
func (q *boundedMemoryQueue) Size() int {
	if q.stopped.Load() {
		return 0
	}
	sizeChan := make(chan int)
	q.eventChan <- sizeChan
	return <-sizeChan
}

func (q *boundedMemoryQueue) Capacity() int {
	return q.capacity
}

func (q *boundedMemoryQueue) IsPersistent() bool {
	return false
}
