// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue // import "go.opentelemetry.io/collector/exporter/exporterqueue"

import (
	"context"
	"errors"
	"sync"
)

var errInvalidSize = errors.New("invalid element size")

type node[T any] struct {
	ctx  context.Context
	data T
	size int64
	next *node[T]
}

type linkedQueue[T any] struct {
	head *node[T]
	tail *node[T]
}

func (l *linkedQueue[T]) push(ctx context.Context, data T, size int64) {
	n := &node[T]{ctx: ctx, data: data, size: size}
	if l.tail == nil {
		l.head = n
		l.tail = n
		return
	}
	l.tail.next = n
	l.tail = n
}

func (l *linkedQueue[T]) pop() (context.Context, T, int64) {
	n := l.head
	l.head = n.next
	if l.head == nil {
		l.tail = nil
	}
	n.next = nil
	return n.ctx, n.data, n.size
}

// sizedQueue is a channel wrapper for sized elements with a capacity set to a total size of all the elements.
// The channel will accept elements until the total size of the elements reaches the capacity.
type sizedQueue[T any] struct {
	sizer sizer[T]
	cap   int64

	mu          sync.Mutex
	hasElements *sync.Cond
	items       *linkedQueue[T]
	size        int64
	stopped     bool
}

// newSizedQueue creates a sized elements channel. Each element is assigned a size by the provided sizer.
// capacity is the capacity of the queue.
func newSizedQueue[T any](capacity int64, sizer sizer[T]) *sizedQueue[T] {
	sq := &sizedQueue[T]{
		sizer: sizer,
		cap:   capacity,
		items: &linkedQueue[T]{},
	}
	sq.hasElements = sync.NewCond(&sq.mu)
	return sq
}

// Offer puts the element into the queue with the given sized if there is enough capacity.
// Returns an error if the queue is full.
func (sq *sizedQueue[T]) Offer(ctx context.Context, el T) error {
	elSize := sq.sizer.Sizeof(el)
	if elSize == 0 {
		return nil
	}

	if elSize <= 0 {
		return errInvalidSize
	}

	sq.mu.Lock()
	defer sq.mu.Unlock()

	if sq.size+elSize > sq.cap {
		return ErrQueueIsFull
	}

	sq.size += elSize
	sq.items.push(ctx, el, elSize)
	// Signal one consumer if any.
	sq.hasElements.Signal()
	return nil
}

// pop removes the element from the queue and returns it.
// The call blocks until there is an item available or the queue is stopped.
// The function returns true when an item is consumed or false if the queue is stopped and emptied.
func (sq *sizedQueue[T]) pop() (context.Context, T, bool) {
	sq.mu.Lock()
	defer sq.mu.Unlock()

	for {
		if sq.size > 0 {
			ctx, el, elSize := sq.items.pop()
			sq.size -= elSize
			return ctx, el, true
		}

		if sq.stopped {
			var el T
			return context.Background(), el, false
		}

		sq.hasElements.Wait()
	}
}

// Shutdown closes the queue channel to initiate draining of the queue.
func (sq *sizedQueue[T]) Shutdown(context.Context) error {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	sq.stopped = true
	sq.hasElements.Broadcast()
	return nil
}

func (sq *sizedQueue[T]) Size() int64 {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	return sq.size
}

func (sq *sizedQueue[T]) Capacity() int64 {
	return sq.cap
}
