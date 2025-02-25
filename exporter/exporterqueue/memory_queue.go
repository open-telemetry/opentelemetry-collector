// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue // import "go.opentelemetry.io/collector/exporter/exporterqueue"

import (
	"context"
	"errors"
	"sync"

	"go.opentelemetry.io/collector/component"
)

var sizeDonePool = sync.Pool{
	New: func() any {
		return &sizeDone{}
	},
}

var errInvalidSize = errors.New("invalid element size")

// memoryQueueSettings defines internal parameters for boundedMemoryQueue creation.
type memoryQueueSettings[T any] struct {
	sizer    sizer[T]
	capacity int64
	blocking bool
}

// memoryQueue is an in-memory implementation of a Queue.
type memoryQueue[T any] struct {
	component.StartFunc
	sizer sizer[T]
	cap   int64

	mu              sync.Mutex
	hasMoreElements *sync.Cond
	hasMoreSpace    *cond
	items           *linkedQueue[T]
	size            int64
	stopped         bool
	blocking        bool
}

// newMemoryQueue creates a sized elements channel. Each element is assigned a size by the provided sizer.
// capacity is the capacity of the queue.
func newMemoryQueue[T any](set memoryQueueSettings[T]) readableQueue[T] {
	sq := &memoryQueue[T]{
		sizer:    set.sizer,
		cap:      set.capacity,
		items:    &linkedQueue[T]{},
		blocking: set.blocking,
	}
	sq.hasMoreElements = sync.NewCond(&sq.mu)
	sq.hasMoreSpace = newCond(&sq.mu)
	return sq
}

// Offer puts the element into the queue with the given sized if there is enough capacity.
// Returns an error if the queue is full.
func (sq *memoryQueue[T]) Offer(ctx context.Context, el T) error {
	elSize := sq.sizer.Sizeof(el)
	if elSize == 0 {
		return nil
	}

	if elSize <= 0 {
		return errInvalidSize
	}

	sq.mu.Lock()
	defer sq.mu.Unlock()

	for sq.size+elSize > sq.cap {
		if !sq.blocking {
			return ErrQueueIsFull
		}
		// Wait for more space or before the ctx is Done.
		if err := sq.hasMoreSpace.Wait(ctx); err != nil {
			return err
		}
	}

	sq.size += elSize
	// Prevent cancellation and deadline to propagate to the context stored in the queue.
	// The grpc/http based receivers will cancel the request context after this function returns.
	sq.items.push(context.WithoutCancel(ctx), el, elSize)
	// Signal one consumer if any.
	sq.hasMoreElements.Signal()
	return nil
}

// Read removes the element from the queue and returns it.
// The call blocks until there is an item available or the queue is stopped.
// The function returns true when an item is consumed or false if the queue is stopped and emptied.
func (sq *memoryQueue[T]) Read(context.Context) (context.Context, T, Done, bool) {
	sq.mu.Lock()
	defer sq.mu.Unlock()

	for {
		if sq.items.hasElements() {
			elCtx, el, elSize := sq.items.pop()
			sd := sizeDonePool.Get().(*sizeDone)
			sd.reset(elSize, sq)
			return elCtx, el, sd, true
		}

		if sq.stopped {
			var el T
			return context.Background(), el, nil, false
		}

		// TODO: Need to change the Queue interface to return an error to allow distinguish between shutdown and context canceled.
		//  Until then use the sync.Cond.
		sq.hasMoreElements.Wait()
	}
}

func (sq *memoryQueue[T]) onDone(elSize int64) {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	sq.size -= elSize
	sq.hasMoreSpace.Signal()
}

// Shutdown closes the queue channel to initiate draining of the queue.
func (sq *memoryQueue[T]) Shutdown(context.Context) error {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	sq.stopped = true
	sq.hasMoreElements.Broadcast()
	return nil
}

func (sq *memoryQueue[T]) Size() int64 {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	return sq.size
}

func (sq *memoryQueue[T]) Capacity() int64 {
	return sq.cap
}

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
	// If tail is nil means list is empty so update both head and tail to point to same element.
	if l.tail == nil {
		l.head = n
		l.tail = n
		return
	}
	l.tail.next = n
	l.tail = n
}

func (l *linkedQueue[T]) hasElements() bool {
	return l.head != nil
}

func (l *linkedQueue[T]) pop() (context.Context, T, int64) {
	n := l.head
	l.head = n.next
	// If it gets to the last element, then update tail as well.
	if l.head == nil {
		l.tail = nil
	}
	n.next = nil
	return n.ctx, n.data, n.size
}

type sizeDone struct {
	size  int64
	queue interface {
		onDone(int64)
	}
}

func (sd *sizeDone) reset(size int64, queue interface{ onDone(int64) }) {
	sd.size = size
	sd.queue = queue
}

func (sd *sizeDone) OnDone(error) {
	defer sizeDonePool.Put(sd)
	sd.queue.onDone(sd.size)
}
