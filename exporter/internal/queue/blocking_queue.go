// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/internal/queue"

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
)

// boundedMemoryQueue blocks insert until the batch containing the request is sent out.
type blockingMemoryQueue[T any] struct {
	component.StartFunc
	*sizedChannel[blockingMemQueueEl[T]]
	sizer Sizer[T]

	mu        sync.Mutex
	nextIndex uint64
	doneCh    map[uint64](chan error)
}

// MemoryQueueSettings defines internal parameters for boundedMemoryQueue creation.
type BlockingMemoryQueueSettings[T any] struct {
	Sizer    Sizer[T]
	Capacity int64
}

// NewBoundedMemoryQueue constructs the new queue of specified capacity, and with an optional
// callback for dropped items (e.g. useful to emit metrics).
func NewBlockingMemoryQueue[T any](set BlockingMemoryQueueSettings[T]) Queue[T] {
	return &blockingMemoryQueue[T]{
		sizedChannel: newSizedChannel[blockingMemQueueEl[T]](set.Capacity, nil, 0),
		sizer:        set.Sizer,
		nextIndex:    0,
		doneCh:       make(map[uint64](chan error)),
	}
}

// Offer is used by the producer to submit new item to the queue. Calling this method on a stopped queue will panic.
func (q *blockingMemoryQueue[T]) Offer(ctx context.Context, req T) error {
	q.mu.Lock()
	index := q.nextIndex
	q.nextIndex++
	done := make(chan error)
	q.doneCh[index] = done

	if err := q.sizedChannel.push(
		blockingMemQueueEl[T]{ctx: ctx, req: req, index: index},
		q.sizer.Sizeof(req),
		nil); err != nil {
		delete(q.doneCh, index)
		q.mu.Unlock()
		return err
	}

	q.mu.Unlock()
	err := <-done
	return err
}

func (q *blockingMemoryQueue[T]) Read(_ context.Context) (uint64, context.Context, T, bool) {
	item, ok := q.sizedChannel.pop(func(el blockingMemQueueEl[T]) int64 { return q.sizer.Sizeof(el.req) })
	return item.index, item.ctx, item.req, ok
}

// OnProcessingFinished should be called to remove the item of the given index from the queue once processing is finished.
// For in memory queue, this function is noop.
func (q *blockingMemoryQueue[T]) OnProcessingFinished(index uint64, err error) {
	q.mu.Lock()
	q.doneCh[index] <- err
	delete(q.doneCh, index)
	q.mu.Unlock()
}

// Shutdown closes the queue channel to initiate draining of the queue.
func (q *blockingMemoryQueue[T]) Shutdown(context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.sizedChannel.shutdown()
	return nil
}

func (q *blockingMemoryQueue[T]) IsBlocking() bool {
	return true
}

type blockingMemQueueEl[T any] struct {
	index uint64
	req   T
	ctx   context.Context
}
