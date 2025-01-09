// Copyright The OpenTelemetry Authors
// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
// SPDX-License-Identifier: Apache-2.0

package exporterqueue // import "go.opentelemetry.io/collector/exporter/exporterqueue"

import (
	"context"

	"go.opentelemetry.io/collector/component"
)

// boundedMemoryQueue implements a producer-consumer exchange similar to a ring buffer queue,
// where the queue is bounded and if it fills up due to slow consumers, the new items written by
// the producer are dropped.
type boundedMemoryQueue[T any] struct {
	component.StartFunc
	*sizedChannel[memQueueEl[T]]
	sizer sizer[T]
}

// memoryQueueSettings defines internal parameters for boundedMemoryQueue creation.
type memoryQueueSettings[T any] struct {
	sizer    sizer[T]
	capacity int64
}

// newBoundedMemoryQueue constructs the new queue of specified capacity, and with an optional
// callback for dropped items (e.g. useful to emit metrics).
func newBoundedMemoryQueue[T any](set memoryQueueSettings[T]) Queue[T] {
	return &boundedMemoryQueue[T]{
		sizedChannel: newSizedChannel[memQueueEl[T]](set.capacity, nil, 0),
		sizer:        set.sizer,
	}
}

// Offer is used by the producer to submit new item to the queue. Calling this method on a stopped queue will panic.
func (q *boundedMemoryQueue[T]) Offer(ctx context.Context, req T) error {
	return q.sizedChannel.push(memQueueEl[T]{ctx: ctx, req: req}, q.sizer.Sizeof(req), nil)
}

func (q *boundedMemoryQueue[T]) Read(_ context.Context) (uint64, context.Context, T, bool) {
	item, ok := q.sizedChannel.pop(func(el memQueueEl[T]) int64 { return q.sizer.Sizeof(el.req) })
	return 0, item.ctx, item.req, ok
}

// OnProcessingFinished should be called to remove the item of the given index from the queue once processing is finished.
// For in memory queue, this function is noop.
func (q *boundedMemoryQueue[T]) OnProcessingFinished(uint64, error) {
}

// Shutdown closes the queue channel to initiate draining of the queue.
func (q *boundedMemoryQueue[T]) Shutdown(context.Context) error {
	q.sizedChannel.shutdown()
	return nil
}

type memQueueEl[T any] struct {
	req T
	ctx context.Context
}
