// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batcher // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/batcher"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

// disabledBatcher is a special-case of Batcher that has no size limit for sending. Any items read from the queue will
// be sent out (asynchronously) immediately regardless of the size.
type disabledBatcher[T any] struct {
	component.StartFunc
	component.ShutdownFunc
	consumeFunc sender.SendFunc[T]
}

func (db *disabledBatcher[T]) Consume(ctx context.Context, req T, done queuebatch.Done) {
	done.OnDone(db.consumeFunc(ctx, req))
}

func newDisabledBatcher[K any](consumeFunc sender.SendFunc[K]) Batcher[K] {
	return &disabledBatcher[K]{consumeFunc: consumeFunc}
}
