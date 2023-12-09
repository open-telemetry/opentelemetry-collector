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

// Queue defines a producer-consumer exchange which can be backed by e.g. the memory-based ring buffer queue
// (boundedMemoryQueue) or via a disk-based queue (persistentQueue)
type Queue[T any] interface {
	component.Component
	// Offer inserts the specified element into this queue if it is possible to do so immediately
	// without violating capacity restrictions. If success returns no error.
	// It returns ErrQueueIsFull if no space is currently available.
	Offer(ctx context.Context, item T) error
	// Consume applies the provided function on the head of queue.
	// The call blocks until there is an item available or the queue is stopped.
	// The function returns true when an item is consumed or false if the queue is stopped.
	Consume(func(ctx context.Context, item T)) bool
	// RequeuingAllowed returns true if the queue expects the consumer to requeue the item in case of temporary failure.
	RequeuingAllowed() bool
}
