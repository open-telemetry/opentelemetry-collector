// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"sync/atomic"
)

type itemsCounter interface {
	ItemsCount() int
}

// QueueCapacityLimiter is an interface to control the capacity of the queue.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type QueueCapacityLimiter[T any] interface {
	// Claim tries to claim capacity for the given element. If the capacity is not available, it returns false.
	Claim(T) bool
	// Release releases capacity for the given queue element.
	Release(T)
	// Size is current size of the queue.
	Size() int
	// Capacity is the maximum capacity of the queue.
	Capacity() int
}

type baseQueueCapacityLimiter struct {
	size     *atomic.Uint64
	capacity uint64
}

func newBaseQueueCapacityLimiter(capacity int) *baseQueueCapacityLimiter {
	return &baseQueueCapacityLimiter{
		size:     &atomic.Uint64{},
		capacity: uint64(capacity),
	}
}

func (bqcl *baseQueueCapacityLimiter) Size() int {
	return int(bqcl.size.Load())
}

func (bqcl *baseQueueCapacityLimiter) Capacity() int {
	return int(bqcl.capacity)
}

// requestCapacityLimiter is a capacity limiter that limits the queue based on the number of requests.
type requestsCapacityLimiter[T any] struct {
	baseQueueCapacityLimiter
}

func NewRequestsCapacityLimiter[T any](capacity int) QueueCapacityLimiter[T] {
	return &requestsCapacityLimiter[T]{
		baseQueueCapacityLimiter: *newBaseQueueCapacityLimiter(capacity),
	}
}

func (rcl *requestsCapacityLimiter[T]) Claim(_ T) bool {
	if rcl.size.Load() >= rcl.capacity {
		return false
	}
	rcl.size.Add(1)
	return true
}

func (rcl *requestsCapacityLimiter[T]) Release(_ T) {
	rcl.size.Add(^uint64(0))
}

// itemsCapacityLimiter is a capacity limiter that limits the queue based on the number of items (e.g. spans, log records).
type itemsCapacityLimiter[T itemsCounter] struct {
	baseQueueCapacityLimiter
}

func NewItemsCapacityLimiter[T itemsCounter](capacity int) QueueCapacityLimiter[T] {
	return &itemsCapacityLimiter[T]{
		baseQueueCapacityLimiter: *newBaseQueueCapacityLimiter(capacity),
	}
}

func (icl *itemsCapacityLimiter[T]) Claim(el T) bool {
	if icl.size.Load() > icl.capacity-uint64(el.ItemsCount()) {
		return false
	}
	icl.size.Add(uint64(el.ItemsCount()))
	return true
}

func (icl *itemsCapacityLimiter[T]) Release(el T) {
	icl.size.Add(^uint64(el.ItemsCount() - 1))
}
