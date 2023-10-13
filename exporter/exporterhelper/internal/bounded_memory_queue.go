// Copyright The OpenTelemetry Authors
// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/collector/component"
)

// queueEvent is an internal event passed in through the event loop channel
// of the queue.
type queueEvent struct {
	request    Request
	stopChan   chan struct{}
	acceptChan chan bool
	done       bool
	sizeChan   chan int
}

// boundedMemoryQueue implements a producer-consumer exchange similar to a ring buffer queue,
// where the queue is bounded and if it fills up due to slow consumers, the new items written by
// the producer are dropped.
type boundedMemoryQueue struct {
	size         int
	eventChan    chan *queueEvent
	items        chan Request
	capacity     uint32
	numConsumers int
	stopped      *atomic.Bool
}

// NewBoundedMemoryQueue constructs the new queue of specified capacity, and with an optional
// callback for dropped items (e.g. useful to emit metrics).
func NewBoundedMemoryQueue(capacity int, numConsumers int) ProducerConsumerQueue {
	return &boundedMemoryQueue{
		items:        make(chan Request, capacity),
		eventChan:    make(chan *queueEvent),
		capacity:     uint32(capacity),
		numConsumers: numConsumers,
		stopped:      &atomic.Bool{},
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
				q.eventChan <- &queueEvent{done: true}
			}
		}()
	}
	startWG.Wait()
	return nil
}

func (q *boundedMemoryQueue) eventLoop() {
	for {
		e := <-q.eventChan
		if e == nil {
			return
		}
		if e.sizeChan != nil {
			e.sizeChan <- q.size
			continue
		}
		if e.done {
			q.size--
			continue
		}
		if q.stopped.Load() && e.stopChan != nil {
			// if we have no consumers, empty the queue.
			if q.numConsumers == 0 {
				for len(q.items) > 0 {
					<-q.items
					q.size--
				}
			}
			if q.size > 0 {
				// we have a stop signal, but there are still elements in the queue.

				// Requeue:
				go func() {
					q.eventChan <- e
				}()
				continue
			}

			close(q.items)
			close(q.eventChan)
			close(e.stopChan)
			return
		}
		if e.stopChan != nil {
			// mark the event loop stopped and requeue to close.
			q.stopped.Store(true)
			go func() {
				q.eventChan <- &queueEvent{stopChan: e.stopChan}
			}()
			continue
		}
		if q.size >= int(q.capacity) || q.capacity == 0 {
			e.acceptChan <- false
			continue
		}
		q.size++
		select {
		case q.items <- e.request:
			e.acceptChan <- true
		default:
			q.size--
			e.acceptChan <- false
		}
	}
}

// Produce is used by the producer to submit new item to the queue. Returns false in case of queue overflow.
func (q *boundedMemoryQueue) Produce(item Request) bool {
	fmt.Println("produce")
	//debug.PrintStack()
	if q.stopped.Load() {
		fmt.Println("stopped")
		return false
	}
	waitForAccept := make(chan bool, 1)
	pipelineItem := &queueEvent{
		request:    item,
		acceptChan: waitForAccept,
	}
	q.eventChan <- pipelineItem
	return <-waitForAccept
}

// Stop stops all consumers, as well as the length reporter if started,
// and releases the items channel. It blocks until all consumers have stopped.
func (q *boundedMemoryQueue) Stop() {
	stopChan := make(chan struct{})
	q.eventChan <- &queueEvent{stopChan: stopChan}
	<-stopChan
}

// Size returns the current size of the queue
func (q *boundedMemoryQueue) Size() int {
	if q.stopped.Load() {
		return 0
	}
	sizeChan := make(chan int)
	q.eventChan <- &queueEvent{sizeChan: sizeChan}
	return <-sizeChan
}

func (q *boundedMemoryQueue) Capacity() int {
	return int(q.capacity)
}

func (q *boundedMemoryQueue) IsPersistent() bool {
	return false
}
