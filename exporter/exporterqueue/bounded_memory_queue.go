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
	*sizedQueue[T]
}

// memoryQueueSettings defines internal parameters for boundedMemoryQueue creation.
type memoryQueueSettings[T any] struct {
	sizer    sizer[T]
	capacity int64
	blocking bool
}

// newBoundedMemoryQueue constructs the new queue of specified capacity, and with an optional
// callback for dropped items (e.g. useful to emit metrics).
func newBoundedMemoryQueue[T any](set memoryQueueSettings[T]) Queue[T] {
	return &boundedMemoryQueue[T]{
		sizedQueue: newSizedQueue[T](set.capacity, set.sizer, set.blocking),
	}
}

func (q *boundedMemoryQueue[T]) Read(context.Context) (uint64, context.Context, T, bool) {
	ctx, req, ok := q.sizedQueue.pop()
	return 0, ctx, req, ok
}

// OnProcessingFinished should be called to remove the item of the given index from the queue once processing is finished.
// For in memory queue, this function is noop.
func (q *boundedMemoryQueue[T]) OnProcessingFinished(uint64, error) {}
