// Copyright The OpenTelemetry Authors
// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
)

var (
	// ErrQueueIsFull is the error returned when an item is offered to the Queue and the queue is full.
	ErrQueueIsFull = errors.New("sending queue is full")
	// ErrQueueIsStopped is the error returned when an item is offered to the Queue and the queue is stopped.
	ErrQueueIsStopped = errors.New("sending queue is stopped")
)

type QueueSettings struct {
	DataType component.DataType
	Callback func(QueueRequest)
}

// Queue defines a producer-consumer exchange which can be backed by e.g. the memory-based ring buffer queue
// (boundedMemoryQueue) or via a disk-based queue (persistentQueue)
type Queue interface {
	// Start starts the queue with a given number of goroutines consuming items from the queue
	// and passing them into the consumer callback.
	Start(ctx context.Context, host component.Host, set QueueSettings) error
	// Offer inserts the specified element into this queue if it is possible to do so immediately
	// without violating capacity restrictions. If success returns no error.
	// It returns ErrQueueIsFull if no space is currently available.
	Offer(ctx context.Context, item any) error
	// Size returns the current Size of the queue
	Size() int
	// Shutdown stops accepting items, and stops all consumers. It blocks until all consumers have stopped.
	Shutdown(ctx context.Context) error
	// Capacity returns the capacity of the queue.
	Capacity() int
}
