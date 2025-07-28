// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

var blockingDonePool = sync.Pool{
	New: func() any {
		return &blockingDone{
			ch: make(chan error, 1),
		}
	},
}

var (
	errInvalidSize  = errors.New("invalid element size")
	errSizeTooLarge = errors.New("element size too large")
)

// memoryQueue is an in-memory implementation of a Queue.
type memoryQueue[T any] struct {
	component.StartFunc
	sizer request.Sizer[T]
	cap   int64

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
func newMemoryQueue[T any](set Settings[T]) readableQueue[T] {
	sq := &memoryQueue[T]{
		sizer:           set.activeSizer(),
		cap:             set.Capacity,
		items:           &linkedQueue[T]{},
		waitForResult:   set.WaitForResult,
		blockOnOverflow: set.BlockOnOverflow,
	}
	sq.hasMoreElements = sync.NewCond(&sq.mu)
	sq.hasMoreSpace = newCond(&sq.mu)
	return sq
}

// Offer puts the element into the queue with the given sized if there is enough capacity.
// Returns an error if the queue is full.
func (mq *memoryQueue[T]) Offer(ctx context.Context, el T) error {
	elSize := mq.sizer.Sizeof(el)
	fmt.Printf("DEBUG: Offer() called with elSize=%d\n", elSize)
	// Ignore empty requests, see https://github.com/open-telemetry/opentelemetry-proto/blob/main/docs/specification.md#empty-telemetry-envelopes
	if elSize == 0 {
		fmt.Printf("DEBUG: Offer() elSize=0, returning nil\n")
		return nil
	}

	if elSize <= 0 {
		fmt.Printf("DEBUG: Offer() elSize<=0, returning errInvalidSize\n")
		return errInvalidSize
	}

	// If element larger than the capacity, will never been able to add it.
	if elSize > mq.cap {
		fmt.Printf("DEBUG: Offer() elSize > capacity, returning errSizeTooLarge\n")
		return errSizeTooLarge
	}

	fmt.Printf("DEBUG: Offer() calling add()\n")
	done, err := mq.add(ctx, el, elSize)
	if err != nil {
		fmt.Printf("DEBUG: Offer() add() returned error: %v\n", err)
		return err
	}

	fmt.Printf("DEBUG: Offer() add() succeeded, waitForResult=%v\n", mq.waitForResult)
	if mq.waitForResult {
		fmt.Printf("DEBUG: Offer() waiting for result from done.ch for blockingDone %p\n", done)
		// Only re-add the blockingDone instance back to the pool if successfully received the
		// message from the consumer which guarantees consumer will not use that anymore,
		// otherwise no guarantee about when the consumer will add the message to the channel so cannot reuse or close.
		select {
		case doneErr := <-done.ch:
			fmt.Printf("DEBUG: Offer() received result from done.ch: %v\n", doneErr)
			blockingDonePool.Put(done)
			return doneErr
		case <-ctx.Done():
			fmt.Printf("DEBUG: Offer() context canceled: %v\n", ctx.Err())
			return ctx.Err()
		}
	}
	fmt.Printf("DEBUG: Offer() waitForResult=false, returning nil\n")
	return nil
}

func (mq *memoryQueue[T]) add(ctx context.Context, el T, elSize int64) (*blockingDone, error) {
	fmt.Printf("DEBUG: add() called with elSize=%d, acquiring lock\n", elSize)
	mq.mu.Lock()
	defer func() {
		fmt.Printf("DEBUG: add() releasing lock\n")
		mq.mu.Unlock()
	}()

	fmt.Printf("DEBUG: add() current size=%d, capacity=%d, would be=%d\n", mq.size, mq.cap, mq.size+elSize)
	blockWaitCount := 0
	for mq.size+elSize > mq.cap {
		blockWaitCount++
		fmt.Printf("DEBUG: add() size check failed (wait #%d), size=%d + elSize=%d > cap=%d\n", 
			blockWaitCount, mq.size, elSize, mq.cap)
		
		if !mq.blockOnOverflow {
			fmt.Printf("DEBUG: add() blockOnOverflow=false, returning ErrQueueIsFull\n")
			return nil, ErrQueueIsFull
		}
		// Wait for more space or before the ctx is Done.
		fmt.Printf("DEBUG: add() calling hasMoreSpace.Wait() - BLOCKING\n")
		if err := mq.hasMoreSpace.Wait(ctx); err != nil {
			fmt.Printf("DEBUG: add() hasMoreSpace.Wait() returned error: %v\n", err)
			return nil, err
		}
		fmt.Printf("DEBUG: add() woke up from hasMoreSpace.Wait(), rechecking size\n")
	}

	fmt.Printf("DEBUG: add() passed size check, adding element\n")
	mq.size += elSize
	done := blockingDonePool.Get().(*blockingDone)
	done.reset(elSize, mq)
	fmt.Printf("DEBUG: add() created blockingDone %p, new size=%d\n", done, mq.size)

	if !mq.waitForResult {
		// Prevent cancellation and deadline to propagate to the context stored in the queue.
		// The grpc/http based receivers will cancel the request context after this function returns.
		ctx = context.WithoutCancel(ctx)
	}

	mq.items.push(ctx, el, done)
	fmt.Printf("DEBUG: add() pushed element to queue, calling hasMoreElements.Broadcast()\n")
	// Signal one consumer if any. Use Broadcast to be more defensive against race conditions.
	mq.hasMoreElements.Broadcast()
	fmt.Printf("DEBUG: add() broadcast completed, returning done %p\n", done)
	return done, nil
}

// Read removes the element from the queue and returns it.
// The call blocks until there is an item available or the queue is stopped.
// The function returns true when an item is consumed or false if the queue is stopped and emptied.
func (mq *memoryQueue[T]) Read(context.Context) (context.Context, T, Done, bool) {
	fmt.Printf("DEBUG: Read() called, acquiring lock\n")
	mq.mu.Lock()
	defer func() {
		fmt.Printf("DEBUG: Read() releasing lock\n")
		mq.mu.Unlock()
	}()

	loopCount := 0
	for {
		loopCount++
		fmt.Printf("DEBUG: Read() loop iteration %d\n", loopCount)
		
		fmt.Printf("DEBUG: Read() checking hasElements: %v\n", mq.items.hasElements())
		if mq.items.hasElements() {
			fmt.Printf("DEBUG: Read() found element, popping from queue\n")
			elCtx, el, done := mq.items.pop()
			fmt.Printf("DEBUG: Read() returning element, queue size now: %d\n", mq.size)
			return elCtx, el, done, true
		}

		fmt.Printf("DEBUG: Read() checking stopped: %v\n", mq.stopped)
		if mq.stopped {
			fmt.Printf("DEBUG: Read() queue stopped, returning false\n")
			var el T
			return context.Background(), el, nil, false
		}

		// TODO: Need to change the Queue interface to return an error to allow distinguish between shutdown and context canceled.
		//  Until then use the sync.Cond.
		// Double-check condition after acquiring lock to prevent lost wakeup
		fmt.Printf("DEBUG: Read() double-checking hasElements: %v, stopped: %v\n", mq.items.hasElements(), mq.stopped)
		if !mq.items.hasElements() && !mq.stopped {
			fmt.Printf("DEBUG: Read() calling hasMoreElements.Wait() - BLOCKING\n")
			mq.hasMoreElements.Wait()
			fmt.Printf("DEBUG: Read() woke up from hasMoreElements.Wait()\n")
		}
	}
}

func (mq *memoryQueue[T]) onDone(bd *blockingDone, err error) {
	fmt.Printf("DEBUG: onDone() called for blockingDone %p with err=%v, acquiring lock\n", bd, err)
	mq.mu.Lock()
	defer func() {
		fmt.Printf("DEBUG: onDone() releasing lock\n")
		mq.mu.Unlock()
	}()
	
	fmt.Printf("DEBUG: onDone() reducing size by %d (from %d to %d)\n", bd.elSize, mq.size, mq.size-bd.elSize)
	mq.size -= bd.elSize
	fmt.Printf("DEBUG: onDone() calling hasMoreSpace.Signal()\n")
	mq.hasMoreSpace.Signal()
	fmt.Printf("DEBUG: onDone() hasMoreSpace.Signal() completed\n")
	
	if mq.waitForResult {
		fmt.Printf("DEBUG: onDone() waitForResult=true, sending err to channel\n")
		// In this case the done will be added back to the queue by the waiter.
		bd.ch <- err
		fmt.Printf("DEBUG: onDone() sent err to channel, returning\n")
		return
	}
	fmt.Printf("DEBUG: onDone() waitForResult=false, putting blockingDone back to pool\n")
	blockingDonePool.Put(bd)
}

// Shutdown closes the queue channel to initiate draining of the queue.
func (mq *memoryQueue[T]) Shutdown(context.Context) error {
	fmt.Printf("DEBUG: Shutdown() called, acquiring lock\n")
	mq.mu.Lock()
	defer func() {
		fmt.Printf("DEBUG: Shutdown() releasing lock\n")
		mq.mu.Unlock()
	}()
	
	fmt.Printf("DEBUG: Shutdown() setting stopped=true\n")
	mq.stopped = true
	fmt.Printf("DEBUG: Shutdown() calling hasMoreElements.Broadcast()\n")
	mq.hasMoreElements.Broadcast()
	// When shutting down, also signal any waiters that they should stop waiting
	// This handles the case where WaitForResult=true and producers are waiting for done.ch
	fmt.Printf("DEBUG: Shutdown() calling hasMoreSpace.Broadcast()\n")
	mq.hasMoreSpace.Broadcast()
	fmt.Printf("DEBUG: Shutdown() completed\n")
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

func (l *linkedQueue[T]) push(ctx context.Context, data T, done Done) {
	n := &node[T]{ctx: ctx, data: data, done: done}
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

func (l *linkedQueue[T]) pop() (context.Context, T, Done) {
	n := l.head
	l.head = n.next
	// If it gets to the last element, then update tail as well.
	if l.head == nil {
		l.tail = nil
	}
	n.next = nil
	return n.ctx, n.data, n.done
}

type blockingDone struct {
	queue interface {
		onDone(*blockingDone, error)
	}
	elSize int64
	ch     chan error
}

func (bd *blockingDone) reset(elSize int64, queue interface{ onDone(*blockingDone, error) }) {
	bd.elSize = elSize
	bd.queue = queue
}

func (bd *blockingDone) OnDone(err error) {
	bd.queue.onDone(bd, err)
}
