// Copyright The OpenTelemetry Authors
// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"go.opentelemetry.io/collector/component"
	"sync/atomic"
)

// boundedMemoryQueue implements a producer-consumer exchange similar to a ring buffer queue,
// where the queue is bounded and if it fills up due to slow consumers, the new items written by
// the producer are dropped.
type boundedMemoryQueue struct {
	component.StartFunc
	stopped *atomic.Bool
	items   chan Request
}

// NewBoundedMemoryQueue constructs the new queue of specified capacity, and with an optional
// callback for dropped items (e.g. useful to emit metrics).
func NewBoundedMemoryQueue(capacity int) Queue {
	return &boundedMemoryQueue{
		items:   make(chan Request, capacity),
		stopped: &atomic.Bool{},
	}
}

// Produce is used by the producer to submit new item to the queue. Returns false in case of queue overflow.
func (q *boundedMemoryQueue) Offer(item Request) bool {
	if q.stopped.Load() {
		return false
	}

	select {
	case q.items <- item:
		return true
	default:
		return false
	}
}

// Poll retrieves and removes the head of this queue.
func (q *boundedMemoryQueue) Poll() <-chan Request {
	return q.items
}

// Shutdown stops all consumers, as well as the length reporter if started,
// and releases the items channel. It blocks until all consumers have stopped.
func (q *boundedMemoryQueue) Shutdown(context.Context) error {
	q.stopped.Store(true) // disable producer
	close(q.items)
	return nil
}

// Size returns the current size of the queue
func (q *boundedMemoryQueue) Size() int {
	return len(q.items)
}

func (q *boundedMemoryQueue) Capacity() int {
	return cap(q.items)
}

func (q *boundedMemoryQueue) IsPersistent() bool {
	return false
}
