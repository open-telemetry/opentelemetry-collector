// Copyright The OpenTelemetry Authors
// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/internal/queue"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
)

var (
	// ErrQueueIsFull is the error returned when an item is offered to the Queue and the queue is full.
	ErrQueueIsFull = errors.New("sending queue is full")
)

// Queue defines a producer-consumer exchange which can be backed by e.g. the memory-based ring buffer queue
// (boundedMemoryQueue) or via a disk-based queue (persistentQueue)
type Queue[T any] interface {
	component.Component
	// Offer inserts the specified element into this queue if it is possible to do so immediately
	// without violating capacity restrictions. If success returns no error.
	// It returns ErrQueueIsFull if no space is currently available.
	Offer(ctx context.Context, item T) error
	// Size returns the current Size of the queue
	Size() int
	// Capacity returns the capacity of the queue.
	Capacity() int
	// Read pulls the next available item from the queue along with its index. Once processing is
	// finished, the index should be called with OnProcessingFinished to clean up the storage.
	// The function blocks until an item is available or if the queue is stopped.
	// Returns false if reading failed or if the queue is stopped.
	Read(context.Context) (uint64, context.Context, T, bool)
	// OnProcessingFinished should be called to remove the item of the given index from the queue once processing is finished.
	OnProcessingFinished(index uint64, consumeErr error)
}

// Sizer is an interface that returns the size of the given element.
type Sizer[T any] interface {
	Sizeof(T) int64
}

// RequestSizer is a Sizer implementation that returns the size of a queue element as one request.
type RequestSizer[T any] struct{}

func (rs *RequestSizer[T]) Sizeof(T) int64 {
	return 1
}
