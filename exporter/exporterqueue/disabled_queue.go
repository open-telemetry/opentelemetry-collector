// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue // import "go.opentelemetry.io/collector/exporter/exporterqueue"

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
)

// boundedQueue blocks insert until the batch containing the request is sent out.
type disabledQueue[T any] struct {
	component.StartFunc
	*sizedQueue[disabledMemQueueEl[T]]

	mu        sync.Mutex
	nextIndex uint64
	doneCh    map[uint64](chan error)
}

type disabledMemQueueEl[T any] struct {
	index uint64
	req   T
}

// QueueSettings defines internal parameters for boundedQueue creation.
type disabledQueueSettings[T any] struct {
	sizer    sizer[T]
	capacity int64
}

type disabledQueueSizer[T any] struct {
	sizer sizer[T]
}

func (s disabledQueueSizer[T]) Sizeof(item disabledMemQueueEl[T]) int64 {
	return s.sizer.Sizeof(item.req)
}

// NewBoundedQueue constructs the new queue of specified capacity, and with an optional
// callback for dropped items (e.g. useful to emit metrics).
func NewDisabledQueue[T any](set disabledQueueSettings[T]) Queue[T] {
	return &disabledQueue[T]{
		sizedQueue: newSizedQueue[disabledMemQueueEl[T]](
			set.capacity,
			disabledQueueSizer[T]{sizer: set.sizer},
			true /*blocking*/),
		doneCh: make(map[uint64](chan error)),
	}
}

// Offer is used by the producer to submit new item to the queue. It will block until OnProcessingFinished is called
// on the request.
func (q *disabledQueue[T]) Offer(ctx context.Context, req T) error {
	q.mu.Lock()
	index := q.nextIndex
	q.nextIndex++
	done := make(chan error)
	q.doneCh[index] = done

	if err := q.sizedQueue.Offer(
		ctx,
		disabledMemQueueEl[T]{req: req, index: index}); err != nil {
		delete(q.doneCh, index)
		q.mu.Unlock()
		return err
	}
	q.mu.Unlock()
	err := <-done
	return err
}

func (q *disabledQueue[T]) Read(_ context.Context) (uint64, context.Context, T, bool) {
	ctx, item, ok := q.sizedQueue.pop()
	return item.index, ctx, item.req, ok
}

// OnProcessingFinished unblocks unblocks offer.
func (q *disabledQueue[T]) OnProcessingFinished(index uint64, err error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.doneCh[index] <- err
	delete(q.doneCh, index)
}
