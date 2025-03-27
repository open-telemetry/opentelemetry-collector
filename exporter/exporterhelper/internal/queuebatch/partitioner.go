// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

// Partitioner is an interface that returns the the partition key of the given element.
type Partitioner[T any] interface {
	GetKey(context.Context, T) string
}

type GetKeyFunc[T any] func(context.Context, T) string

func (f GetKeyFunc[T]) GetKey(ctx context.Context, t T) string {
	return f(ctx, t)
}

type BasePartitioner struct {
	GetKeyFunc[request.Request]
}

func NewPartitioner(
	getKeyFunc func(ctx context.Context,
		req request.Request) string,
) Partitioner[request.Request] {
	return &BasePartitioner{
		GetKeyFunc: getKeyFunc,
	}
}
