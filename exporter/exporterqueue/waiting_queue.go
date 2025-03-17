// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue // import "go.opentelemetry.io/collector/exporter/exporterqueue"

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
)

var waitingDonePool = sync.Pool{
	New: func() any {
		return &waitingDone{ch: make(chan error, 1)}
	},
}

func newWaitingQueue[T any](sizer sizer[T], capacity int64, blocking bool, consumeFunc ConsumeFunc[T]) Queue[T] {
	wq := &waitingQueue[T]{
		consumeFunc: consumeFunc,
		sizer:       sizer,
		capacity:    capacity,
		blocking:    blocking,
	}
	wq.hasMoreSpace = newCond(&wq.mu)
	return wq
}

type waitingQueue[T any] struct {
	component.StartFunc
	component.ShutdownFunc
	consumeFunc ConsumeFunc[T]
	sizer       sizer[T]
	capacity    int64

	mu           sync.Mutex
	hasMoreSpace *cond
	size         int64
	blocking     bool
}

func (wq *waitingQueue[T]) Offer(ctx context.Context, el T) error {
	elSize := wq.sizer.Sizeof(el)
	// Ignore empty requests.
	if elSize == 0 {
		return nil
	}

	if elSize < 0 {
		return errInvalidSize
	}

	if err := wq.add(ctx, elSize); err != nil {
		return err
	}

	done := waitingDonePool.Get().(*waitingDone)
	done.queue = wq
	done.elSize = elSize
	wq.consumeFunc(ctx, el, done)
	// Only re-add the waitingDone instance back to the pool if successfully received the
	// message from the consumer which guarantees consumer will not use that anymore,
	// otherwise no guarantee about when the consumer will add the message to the channel so cannot reuse or close.
	select {
	case doneErr := <-done.ch:
		waitingDonePool.Put(done)
		return doneErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (wq *waitingQueue[T]) add(ctx context.Context, elSize int64) error {
	wq.mu.Lock()
	defer wq.mu.Unlock()

	for wq.size+elSize > wq.capacity {
		if !wq.blocking {
			return ErrQueueIsFull
		}
		// Wait for more space or before the ctx is Done.
		if err := wq.hasMoreSpace.Wait(ctx); err != nil {
			return err
		}
	}
	wq.size += elSize
	return nil
}

func (wq *waitingQueue[T]) onDone(elSize int64) {
	wq.mu.Lock()
	defer wq.mu.Unlock()
	wq.size -= elSize
	wq.hasMoreSpace.Signal()
}

// Size returns the current number of blocked requests waiting to be processed.
func (wq *waitingQueue[T]) Size() int64 {
	wq.mu.Lock()
	defer wq.mu.Unlock()
	return wq.size
}

// Capacity returns the capacity of this queue, which is 0 that means no bounds.
func (wq *waitingQueue[T]) Capacity() int64 {
	return wq.capacity
}

type waitingDone struct {
	queue interface {
		onDone(int64)
	}
	elSize int64
	ch     chan error
}

func (d *waitingDone) OnDone(err error) {
	d.queue.onDone(d.elSize)
	d.ch <- err
}
