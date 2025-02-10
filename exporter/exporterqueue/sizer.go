// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue // import "go.opentelemetry.io/collector/exporter/exporterqueue"

// sizer is an interface that returns the size of the given element.
type sizer[T any] interface {
	Sizeof(T) int64
}

// requestSizer is a sizer implementation that returns the size of a queue element as one request.
type requestSizer[T any] struct{}

func (rs *requestSizer[T]) Sizeof(T) int64 {
	return 1
}
