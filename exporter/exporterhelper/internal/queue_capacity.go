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
type QueueCapacityLimiter[T any] interface {
	// Capacity is the maximum capacity of the queue.
	Capacity() int
	// Size returns the current size of the queue.
	Size() int
	// claim tries to claim capacity for the given element. If the capacity is not available, it returns false.
	claim(T) bool
	// release releases capacity for the given queue element.
	release(T)
	// sizeOf returns the size of the given element.
	sizeOf(T) uint64
}

type baseCapacityLimiter[T any] struct {
	used *atomic.Uint64
	cap  uint64
}

func newBaseCapacityLimiter[T any](capacity int) baseCapacityLimiter[T] {
	return baseCapacityLimiter[T]{
		used: &atomic.Uint64{},
		cap:  uint64(capacity),
	}
}

func (bcl *baseCapacityLimiter[T]) Capacity() int {
	return int(bcl.cap)
}

func (bcl *baseCapacityLimiter[T]) Size() int {
	return int(bcl.used.Load())
}

// nolint:unused false positive https://github.com/dominikh/go-tools/issues/1440
func (bcl *baseCapacityLimiter[T]) claim(size uint64) bool {
	if bcl.used.Add(size) > bcl.cap {
		bcl.release(size)
		return false
	}
	return true
}

// nolint:unused false positive https://github.com/dominikh/go-tools/issues/1440
func (bcl *baseCapacityLimiter[T]) release(size uint64) {
	bcl.used.Add(^(size - 1))
}

// itemsCapacityLimiter is a capacity limiter that limits the queue based on the number of items (e.g. spans, log records).
type itemsCapacityLimiter[T itemsCounter] struct {
	baseCapacityLimiter[T]
}

func NewItemsCapacityLimiter[T itemsCounter](capacity int) QueueCapacityLimiter[T] {
	return &itemsCapacityLimiter[T]{baseCapacityLimiter: newBaseCapacityLimiter[T](capacity)}
}

func (icl *itemsCapacityLimiter[T]) claim(el T) bool {
	return icl.baseCapacityLimiter.claim(uint64(el.ItemsCount()))
}

// nolint:unused false positive https://github.com/dominikh/go-tools/issues/1440
func (icl *itemsCapacityLimiter[T]) release(el T) {
	icl.baseCapacityLimiter.release(uint64(el.ItemsCount()))
}

// nolint:unused false positive https://github.com/dominikh/go-tools/issues/1440
func (icl *itemsCapacityLimiter[T]) sizeOf(el T) uint64 {
	return uint64(el.ItemsCount())
}

// requestsCapacityLimiter is a capacity limiter that limits the queue based on the number of requests.
type requestsCapacityLimiter[T any] struct {
	baseCapacityLimiter[T]
}

func NewRequestsCapacityLimiter[T any](capacity int) QueueCapacityLimiter[T] {
	return &requestsCapacityLimiter[T]{baseCapacityLimiter: newBaseCapacityLimiter[T](capacity)}
}

// nolint:unused false positive https://github.com/dominikh/go-tools/issues/1440
func (rcl *requestsCapacityLimiter[T]) claim(T) bool {
	return rcl.baseCapacityLimiter.claim(1)
}

// nolint:unused false positive https://github.com/dominikh/go-tools/issues/1440
func (rcl *requestsCapacityLimiter[T]) release(T) {
	rcl.baseCapacityLimiter.release(1)
}

// nolint:unused false positive https://github.com/dominikh/go-tools/issues/1440
func (rcl *requestsCapacityLimiter[T]) sizeOf(T) uint64 {
	return 1
}
