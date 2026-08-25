// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

// disabledBatcher is a special-case of Batcher that has no size limit for sending. Any items read from the queue will
// be sent out (asynchronously) immediately regardless of the size.
type disabledBatcher[T any] struct {
	component.StartFunc
	component.ShutdownFunc
	consumeFunc sender.SendFunc[T]
}

func (db *disabledBatcher[T]) Consume(ctx context.Context, req T, done queue.Done) {
	done.OnDone(db.consumeFunc(ctx, req))
}

func newDisabledBatcher[T any](consumeFunc sender.SendFunc[T]) Batcher[T] {
	return &disabledBatcher[T]{consumeFunc: consumeFunc}
}
