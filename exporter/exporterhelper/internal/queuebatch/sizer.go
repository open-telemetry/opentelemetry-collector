// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

// Sizer is an interface that returns the size of the given element.
type Sizer[T any] interface {
	Sizeof(T) int64
}

type SizeofFunc[T any] func(T) int64

func (f SizeofFunc[T]) Sizeof(t T) int64 {
	return f(t)
}

// RequestsSizer is a Sizer implementation that returns the size of a queue element as one request.
type RequestsSizer[T any] struct{}

func (rs RequestsSizer[T]) Sizeof(T) int64 {
	return 1
}
