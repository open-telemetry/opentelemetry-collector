// Copyright The OpenTelemetry Authors
// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

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
	waitEnabled  bool
	waitTimeout  time.Duration
	stoppedChan  chan int
}

// NewBoundedMemoryQueue constructs the new queue of specified capacity, and with an optional
// callback for dropped items (e.g. useful to emit metrics).
func NewBoundedMemoryQueue(capacity int, numConsumers int, waitEnabled bool, waitTimeout time.Duration) ProducerConsumerQueue {
	return &boundedMemoryQueue{
		items:        make(chan Request, capacity),
		stopped:      &atomic.Bool{},
		size:         &atomic.Uint32{},
		capacity:     uint32(capacity),
		numConsumers: numConsumers,
		errCh:        make(chan error, numConsumers),
		waitEnabled:  waitEnabled,
		waitTimeout:  waitTimeout,
		stoppedChan:  make(chan int, 1),
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
				if q.stopped.Load() {
					q.stoppedChan <- 1
					return
				}
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
	if !q.waitEnabled && q.size.Load() >= q.capacity {
		return false
	} else if q.waitEnabled {
		return q.produceAndWait(item)
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

// Same as Produce but waits for response before queuing the next item
func (q *boundedMemoryQueue) produceAndWait(item Request) bool {
	timer := time.NewTimer(q.waitTimeout)
	if q.stopped.Load() {
		return false
	}

	select {
	case <-timer.C:
		return false
	case <-q.stoppedChan:
		// signalled that Stop() was called so close q.items.
		close(q.items)
		return false
	case q.items <- item:
		q.size.Add(1)
		return true
	}
}

// GetErrCh gets the channel that stores responses for sent requests.
func (q *boundedMemoryQueue) GetErrCh() chan error {
	return q.errCh
}

// Stop stops all consumers, as well as the length reporter if started,
// and releases the items channel. It blocks until all consumers have stopped.
func (q *boundedMemoryQueue) Stop() {
	q.stopped.Store(true) // disable producer
	// if waiting is enabled need to signal to senders that Stop()
	// has been called and q.items needs to be gracefully closed.
	// q.stoppedChan <- 1
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
