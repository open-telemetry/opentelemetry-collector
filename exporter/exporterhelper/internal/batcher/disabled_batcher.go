// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batcher // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/batcher"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
)

// disabledBatcher is a special-case of Batcher that has no size limit for sending. Any items read from the queue will
// be sent out (asynchronously) immediately regardless of the size.
type disabledBatcher[T any] struct {
	component.StartFunc
	component.ShutdownFunc
	exportFunc func(context.Context, T) error
}

func (db *disabledBatcher[T]) Consume(ctx context.Context, req T, done exporterqueue.Done) {
	done.OnDone(db.exportFunc(ctx, req))
}

func newDisabledBatcher(exportFunc func(ctx context.Context, req request.Request) error) Batcher {
	return &disabledBatcher[request.Request]{exportFunc: exportFunc}
}
