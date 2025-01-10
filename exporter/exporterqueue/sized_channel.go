// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue // import "go.opentelemetry.io/collector/exporter/exporterqueue"

import (
	"errors"
	"sync/atomic"
)

var errInvalidSize = errors.New("invalid element size")

// sizedChannel is a channel wrapper for sized elements with a capacity set to a total size of all the elements.
// The channel will accept elements until the total size of the elements reaches the capacity.
type sizedChannel[T any] struct {
	sizer sizer[T]
	used  *atomic.Int64
	ch    chan T
}

// newSizedChannel creates a sized elements channel. Each element is assigned a positive size by the provided sizer.
// capacity is the total capacity of the queue.
func newSizedChannel[T any](capacity int64, sizer sizer[T]) *sizedChannel[T] {
	return &sizedChannel[T]{
		sizer: sizer,
		used:  &atomic.Int64{},
		ch:    make(chan T, capacity),
	}
}

// push puts the element into the queue with the given sized if there is enough capacity.
// Returns an error if the queue is full.
func (vcq *sizedChannel[T]) push(el T) error {
	elSize := vcq.sizer.Sizeof(el)
	if elSize <= 0 {
		return errInvalidSize
	}
	if vcq.used.Add(elSize) > int64(cap(vcq.ch)) {
		vcq.used.Add(-elSize)
		return ErrQueueIsFull
	}

	vcq.ch <- el
	return nil
}

// pop removes the element from the queue and returns it.
// The call blocks until there is an item available or the queue is stopped.
// The function returns true when an item is consumed or false if the queue is stopped and emptied.
func (vcq *sizedChannel[T]) pop() (T, bool) {
	el, ok := <-vcq.ch
	if !ok {
		return el, false
	}

	vcq.used.Add(-vcq.sizer.Sizeof(el))
	return el, true
}

// shutdown closes the queue channel to initiate draining of the queue.
func (vcq *sizedChannel[T]) shutdown() {
	close(vcq.ch)
}

func (vcq *sizedChannel[T]) Size() int {
	return int(vcq.used.Load())
}

func (vcq *sizedChannel[T]) Capacity() int {
	return cap(vcq.ch)
}
