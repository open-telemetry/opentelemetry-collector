// Copyright The OpenTelemetry Authors
// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/internal/queue"

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
)

// blockingMemoryQueue implements a producer-consumer exchange similar to a ring buffer queue,
// where the queue is blocking and if it fills up due to slow consumers, the new items written by
// the producer are dropped.
type blockingMemoryQueue[T any] struct {
	component.StartFunc
	*sizedChannel[blockingMemQueueEl[T]]
	sizer Sizer[T]

	mu        sync.Mutex
	nextIdx   uint64
	idxToChan map[uint64](chan error)
}

// MemoryQueueSettings defines internal parameters for blockingMemoryQueue creation.
type BlockingMemoryQueueSettings[T any] struct {
	Sizer    Sizer[T]
	Capacity int64
}

// NewblockingMemoryQueue constructs the new queue of specified capacity, and with an optional
// callback for dropped items (e.g. useful to emit metrics).
func NewBlockingMemoryQueue[T any](set BlockingMemoryQueueSettings[T]) Queue[T] {
	return &blockingMemoryQueue[T]{
		sizedChannel: newSizedChannel[blockingMemQueueEl[T]](set.Capacity, nil, 0),
		sizer:        set.Sizer,
	}
}

// Offer is used by the producer to submit new item to the queue. Calling this method on a stopped queue will panic.
func (q *blockingMemoryQueue[T]) Offer(ctx context.Context, req T) error {
	result := make(chan error, 1)
	q.sizedChannel.push(blockingMemQueueEl[T]{ctx: ctx, req: req, result: result}, q.sizer.Sizeof(req), nil)
	err, _ := <-result
	return err
}

func (q *blockingMemoryQueue[T]) Read(_ context.Context) (uint64, context.Context, T, bool) {
	item, ok := q.sizedChannel.pop(func(el blockingMemQueueEl[T]) int64 { return q.sizer.Sizeof(el.req) })
	q.mu.Lock()
	defer q.mu.Unlock()
	idx := q.nextIdx
	q.nextIdx += 1
	q.idxToChan[idx] = item.result
	return idx, item.ctx, item.req, ok
}

// Should be called to remove the item of the given index from the queue once processing is finished.
// For in memory queue, this function is noop.
func (q *blockingMemoryQueue[T]) OnProcessingFinished(idx uint64, err error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.idxToChan[idx] <- err
	delete(q.idxToChan, idx)
}

// Shutdown closes the queue channel to initiate draining of the queue.
func (q *blockingMemoryQueue[T]) Shutdown(context.Context) error {
	q.sizedChannel.shutdown()
	return nil
}

type blockingMemQueueEl[T any] struct {
	req    T
	ctx    context.Context
	result chan error
	idx    uint64
}
