// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/internal/queue"

import "sync/atomic"

// sizedElementsChannel is a channel wrapper for sized elements with a capacity set to a total size of all the elements.
// The channel will accept elements until the total size of the elements reaches the capacity.
type sizedElementsChannel[T any] struct {
	used *atomic.Int64

	// We need to store the capacity in a separate field because the capacity of the channel can be higher.
	// It happens when we restore a persistent queue from a disk that is bigger than the pre-configured capacity.
	cap int64
	ch  chan T
}

type sizedElementsChannelOption[T any] func(*sizedElementsChannel[T])

// withPreloadElements puts the elements into the queue with the given size. It's used by the persistent queue to
// initialize the queue with the elements recovered from the disk.
func withPreloadElements[T any](els []T, totalSize int64) sizedElementsChannelOption[T] {
	return func(vcq *sizedElementsChannel[T]) {
		vcq.used.Store(totalSize)
		capacity := int(vcq.cap)
		if capacity < len(els) {
			capacity = len(els)
		}
		vcq.ch = make(chan T, capacity)
		for _, el := range els {
			vcq.ch <- el
		}
	}
}

// newSizedElementsChannel creates a sized elements channel. Each element is assigned a size by the provided sizer.
// chanCapacity is the capacity of the underlying channel which usually should be equal to the capacity of the queue to
// avoid blocking the producer.
func newSizedElementsChannel[T any](capacity int, opts ...sizedElementsChannelOption[T]) *sizedElementsChannel[T] {
	sech := &sizedElementsChannel[T]{
		used: &atomic.Int64{},
		cap:  int64(capacity),
	}
	for _, opt := range opts {
		opt(sech)
	}
	if sech.ch == nil {
		sech.ch = make(chan T, capacity)
	}
	return sech
}

// enqueue puts the element into the queue with the given sized if there is enough capacity.
// Returns an error if the queue is full. The callback is called before the element is committed to the queue.
// If the callback returns an error, the element is not put into the queue and the error is returned.
func (vcq *sizedElementsChannel[T]) enqueue(el T, size int64, callback func() error) error {
	if vcq.used.Add(size) > vcq.cap {
		vcq.used.Add(-size)
		return ErrQueueIsFull
	}
	if callback != nil {
		if err := callback(); err != nil {
			vcq.used.Add(-size)
			return err
		}
	}
	vcq.ch <- el
	return nil
}

// dequeue removes the element from the queue and returns it.
// The call blocks until there is an item available or the queue is stopped.
// The function returns true when an item is consumed or false if the queue is stopped and emptied.
// The callback is called before the element is removed from the queue. It must return the size of the element.
func (vcq *sizedElementsChannel[T]) dequeue(callback func(T) (size int64)) (T, bool) {
	el, ok := <-vcq.ch
	if !ok {
		return el, false
	}

	size := callback(el)

	// The used size and the channel size might be not in sync if the queue in case it's restored from the disk.
	// In that case we need to make sure it doesn't go below 0.
	if vcq.used.Add(-size) < 0 {
		vcq.used.Store(0)
	}
	return el, true
}

// syncSize updates the used size to 0 if the queue is empty.
// The caller must ensure that this call is not called concurrently with enqueue.
// It's used by the persistent queue to ensure the used value correctly reflects the reality which may not be always
// the case in case if the queue size is restored from the disk after a crash.
func (vcq *sizedElementsChannel[T]) syncSize() {
	if len(vcq.ch) == 0 {
		vcq.used.Store(0)
	}
}

// shutdown closes the queue channel to initiate draining of the queue.
func (vcq *sizedElementsChannel[T]) shutdown() {
	close(vcq.ch)
}

func (vcq *sizedElementsChannel[T]) Size() int {
	return int(vcq.used.Load())
}

func (vcq *sizedElementsChannel[T]) Capacity() int {
	return int(vcq.cap)
}
