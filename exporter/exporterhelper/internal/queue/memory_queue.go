// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"
	"errors"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

var (
	errInvalidSize  = errors.New("invalid element size")
	errSizeTooLarge = errors.New("element size too large")
)

// memoryQueue is an in-memory implementation of a Queue.
type memoryQueue[T request.Request] struct {
	component.StartFunc
	refCounter ReferenceCounter[T]
	sizer      request.Sizer
	cap        int64
	donePool   sync.Pool

	mu              sync.Mutex
	hasMoreElements *sync.Cond
	hasMoreSpace    *cond
	items           *linkedQueue[T]
	size            int64
	stopped         bool
	waitForResult   bool
	blockOnOverflow bool
}

// newMemoryQueue creates a sized elements channel. Each element is assigned a size by the provided sizer.
// capacity is the capacity of the queue.
func newMemoryQueue[T request.Request](set Settings[T]) readableQueue[T] {
	sq := &memoryQueue[T]{
		refCounter:      set.ReferenceCounter,
		sizer:           request.NewSizer(set.SizerType),
		cap:             set.Capacity,
		items:           &linkedQueue[T]{},
		waitForResult:   set.WaitForResult,
		blockOnOverflow: set.BlockOnOverflow,
	}
	sq.donePool.New = func() any { return &blockingDone[T]{ch: make(chan error, 1)} }
	sq.hasMoreElements = sync.NewCond(&sq.mu)
	sq.hasMoreSpace = newCond(&sq.mu)
	return sq
}

// Offer puts the element into the queue with the given sized if there is enough capacity.
// Returns an error if the queue is full.
func (mq *memoryQueue[T]) Offer(ctx context.Context, el T) error {
	elSize := mq.sizer.Sizeof(el)
	// Ignore empty requests, see https://github.com/open-telemetry/opentelemetry-proto/blob/main/docs/specification.md#empty-telemetry-envelopes
	if elSize == 0 {
		return nil
	}

	if elSize <= 0 {
		return errInvalidSize
	}

	// If element larger than the capacity, will never been able to add it.
	if elSize > mq.cap {
		return errSizeTooLarge
	}

	if mq.refCounter != nil {
		mq.refCounter.Ref(el)
	}

	n, done, err := mq.add(ctx, el, elSize)
	if err != nil {
		// Unref in case of an error since there will not be any async worker to pick it up.
		if mq.refCounter != nil {
			mq.refCounter.Unref(el)
		}
		return err
	}

	if mq.waitForResult {
		// Only re-add the blockingDone instance back to the pool if successfully received the
		// message from the consumer which guarantees consumer will not use that anymore,
		// otherwise no guarantee about when the consumer will add the message to the channel so cannot reuse or close.
		select {
		case doneErr := <-done.ch:
			mq.donePool.Put(done)
			return doneErr
		case <-ctx.Done():
			if mq.abandon(n, done) {
				// No consumer will pick the element up, so onDone never runs for it.
				mq.releaseElement(done)
				return ctx.Err()
			}
			// A consumer already took the element, so it is going to be delivered either way. Report
			// the actual outcome rather than a failure for data that may have been sent. The element
			// may have been merged into a batch, whose done only reports after that batch was sent.
			return <-done.ch
		}
	}
	return nil
}

// abandon gives up on an element whose caller is no longer waiting for it. It reports false if a
// consumer already took the element, in which case it keeps being processed.
func (mq *memoryQueue[T]) abandon(n *node[T], bd *blockingDone[T]) bool {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	if !queued(n) {
		return false
	}
	// Leave the element in place for Read to drop, which keeps this O(1), and give the capacity back
	// now so that a producer blocked on overflow does not wait for a consumer to get here.
	bd.abandoned = true
	mq.freeCapacity(bd)
	return true
}

// freeCapacity gives back the space taken by the element. Callers MUST hold the mutex.
func (mq *memoryQueue[T]) freeCapacity(bd *blockingDone[T]) {
	mq.size -= bd.elSize
	mq.hasMoreSpace.Signal()
}

// releaseElement drops the queue's reference to the element the done reports for.
func (mq *memoryQueue[T]) releaseElement(bd *blockingDone[T]) {
	if mq.refCounter != nil {
		mq.refCounter.Unref(bd.el)
	}
	var zero T
	bd.el = zero
}

func (mq *memoryQueue[T]) add(ctx context.Context, el T, elSize int64) (*node[T], *blockingDone[T], error) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	for mq.size+elSize > mq.cap {
		if !mq.blockOnOverflow {
			return nil, nil, ErrQueueIsFull
		}
		// Wait for more space or before the ctx is Done.
		if err := mq.hasMoreSpace.Wait(ctx); err != nil {
			return nil, nil, err
		}
	}

	mq.size += elSize
	done := mq.donePool.Get().(*blockingDone[T])
	done.reset(el, elSize, mq)

	if !mq.waitForResult {
		// Prevent cancellation and deadline to propagate to the context stored in the queue.
		// The grpc/http based receivers will cancel the request context after this function returns.
		ctx = context.WithoutCancel(ctx)
	}

	n := mq.items.push(ctx, el, done)
	// Signal one consumer if any.
	mq.hasMoreElements.Signal()
	return n, done, nil
}

// Read removes the element from the queue and returns it.
// The call blocks until there is an item available or the queue is stopped.
// The function returns true when an item is consumed or false if the queue is stopped and emptied.
func (mq *memoryQueue[T]) Read(context.Context) (context.Context, T, Done, bool) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	for {
		if mq.items.hasElements() {
			// An abandoned element must not reach a consumer, and nothing here may read the request
			// itself, since the caller that gave up on it has already released it.
			if mq.waitForResult && mq.items.head.done.(*blockingDone[T]).abandoned {
				mq.items.pop()
				continue
			}
			elCtx, el, done := mq.items.pop()
			return elCtx, el, done, true
		}

		if mq.stopped {
			var el T
			return context.Background(), el, nil, false
		}

		// TODO: Need to change the Queue interface to return an error to allow distinguish between shutdown and context canceled.
		//  Until then use the sync.Cond.
		mq.hasMoreElements.Wait()
	}
}

func (mq *memoryQueue[T]) onDone(bd *blockingDone[T], err error) {
	mq.releaseElement(bd)

	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.freeCapacity(bd)
	if mq.waitForResult {
		// In this case the done will be added back to the queue by the waiter.
		bd.ch <- err
		return
	}
	mq.donePool.Put(bd)
}

// Shutdown closes the queue channel to initiate draining of the queue.
func (mq *memoryQueue[T]) Shutdown(context.Context) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.stopped = true
	mq.hasMoreElements.Broadcast()
	return nil
}

func (mq *memoryQueue[T]) Size() int64 {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	return mq.size
}

func (mq *memoryQueue[T]) Capacity() int64 {
	return mq.cap
}

type node[T any] struct {
	ctx  context.Context
	data T
	done Done
	next *node[T]
}

type linkedQueue[T any] struct {
	head *node[T]
	tail *node[T]
}

func (l *linkedQueue[T]) push(ctx context.Context, data T, done Done) *node[T] {
	n := &node[T]{ctx: ctx, data: data, done: done}
	// If tail is nil means list is empty so update both head and tail to point to same element.
	if l.tail == nil {
		l.head = n
		l.tail = n
		return n
	}
	l.tail.next = n
	l.tail = n
	return n
}

func (l *linkedQueue[T]) hasElements() bool {
	return l.head != nil
}

func (l *linkedQueue[T]) pop() (context.Context, T, Done) {
	n := l.head
	l.head = n.next
	// If it gets to the last element, then update tail as well.
	if l.head == nil {
		l.tail = nil
	}
	// Point the node at itself to record that it left the queue. A node that is still queued never
	// does, so queued(n) can tell the two apart without an extra field.
	n.next = n
	return n.ctx, n.data, n.done
}

// queued reports whether the node is still waiting in the queue, as opposed to having been handed
// to a consumer or dropped.
func queued[T any](n *node[T]) bool {
	return n.next != n
}

type blockingDone[T request.Request] struct {
	queue *memoryQueue[T]
	// el is the element this done reports for, kept so that the queue can release its reference to it.
	el     T
	elSize int64
	// abandoned tells Read to drop the element instead of handing it to a consumer, because the
	// caller that was waiting for it has given up. Only ever set while the element is still queued,
	// so an abandoned element never reaches onDone.
	abandoned bool
	ch        chan error
}

func (bd *blockingDone[T]) reset(el T, elSize int64, queue *memoryQueue[T]) {
	bd.el = el
	bd.elSize = elSize
	bd.abandoned = false
	bd.queue = queue
}

func (bd *blockingDone[T]) OnDone(err error) {
	bd.queue.onDone(bd, err)
}
