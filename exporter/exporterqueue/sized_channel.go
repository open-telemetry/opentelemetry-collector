// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue // import "go.opentelemetry.io/collector/exporter/exporterqueue"

import "sync/atomic"

// sizedChannel is a channel wrapper for sized elements with a capacity set to a total size of all the elements.
// The channel will accept elements until the total size of the elements reaches the capacity.
type sizedChannel[T any] struct {
	sizer sizer[T]
	used  *atomic.Int64

	// We need to store the capacity in a separate field because the capacity of the channel can be higher.
	// It happens when we restore a persistent queue from a disk that is bigger than the pre-configured capacity.
	cap int64
	ch  chan T
}

// newSizedChannel creates a sized elements channel. Each element is assigned a size by the provided sizer.
// chanCapacity is the capacity of the underlying channel which usually should be equal to the capacity of the queue to
// avoid blocking the producer. Optionally, the channel can be preloaded with the elements and their total size.
func newSizedChannel[T any](capacity int64, sizer sizer[T]) *sizedChannel[T] {
	used := &atomic.Int64{}

	ch := make(chan T, capacity)
	return &sizedChannel[T]{
		sizer: sizer,
		used:  used,
		cap:   capacity,
		ch:    ch,
	}
}

// push puts the element into the queue with the given sized if there is enough capacity.
// Returns an error if the queue is full. The callback is called before the element is committed to the queue.
// If the callback returns an error, the element is not put into the queue and the error is returned.
// The size is the size of the element MUST be positive.
func (vcq *sizedChannel[T]) push(el T) error {
	elSize := vcq.sizer.Sizeof(el)
	if vcq.used.Add(elSize) > vcq.cap {
		vcq.used.Add(-elSize)
		return ErrQueueIsFull
	}

	select {
	// for persistent queue implementation, channel len can be out of sync with used size. Attempt to put it
	// into the channel. If it is full, simply returns ErrQueueIsFull error. This prevents potential deadlock issues.
	case vcq.ch <- el:
		return nil
	default:
		vcq.used.Add(-elSize)
		return ErrQueueIsFull
	}
}

// pop removes the element from the queue and returns it.
// The call blocks until there is an item available or the queue is stopped.
// The function returns true when an item is consumed or false if the queue is stopped and emptied.
// The callback is called before the element is removed from the queue. It must return the size of the element.
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
	return int(vcq.cap)
}
