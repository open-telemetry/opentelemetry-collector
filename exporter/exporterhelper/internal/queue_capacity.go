// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"sync/atomic"
)

type itemsCounter interface {
	ItemsCount() int
}

// Sizer is an interface that returns the size of the given element.
type Sizer[T any] interface {
	SizeOf(T) uint64
}

// ItemsSizer is a Sizer implementation that returns the size of a queue element as the number of items it contains.
type ItemsSizer[T itemsCounter] struct{}

func (is *ItemsSizer[T]) SizeOf(el T) uint64 {
	return uint64(el.ItemsCount())
}

// RequestSizer is a Sizer implementation that returns the size of a queue element as one request.
type RequestSizer[T any] struct{}

func (rs *RequestSizer[T]) SizeOf(T) uint64 {
	return 1
}

type queueCapacityLimiter[T any] struct {
	used *atomic.Uint64
	cap  uint64
	sz   Sizer[T]
}

func (bcl queueCapacityLimiter[T]) Capacity() int {
	return int(bcl.cap)
}

func (bcl queueCapacityLimiter[T]) Size() int {
	return int(bcl.used.Load())
}

func (bcl queueCapacityLimiter[T]) claim(el T) bool {
	size := bcl.sizeOf(el)
	if bcl.used.Add(size) > bcl.cap {
		bcl.releaseSize(size)
		return false
	}
	return true
}

func (bcl queueCapacityLimiter[T]) release(el T) {
	bcl.releaseSize(bcl.sizeOf(el))
}

func (bcl queueCapacityLimiter[T]) releaseSize(size uint64) {
	bcl.used.Add(^(size - 1))
}

func (bcl queueCapacityLimiter[T]) sizeOf(el T) uint64 {
	return bcl.sz.SizeOf(el)
}

func newQueueCapacityLimiter[T any](sizer Sizer[T], capacity int) *queueCapacityLimiter[T] {
	return &queueCapacityLimiter[T]{
		used: &atomic.Uint64{},
		cap:  uint64(capacity),
		sz:   sizer,
	}
}
