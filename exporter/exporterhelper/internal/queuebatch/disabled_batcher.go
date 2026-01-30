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
	// Check if the producer's context was cancelled before processing.
	// This prevents exporting data when the caller already received a timeout error,
	// which would cause confusing observability results and potential duplicate data
	// if the caller retries.
	if ctx.Err() != nil {
		done.OnDone(ctx.Err())
		return
	}
	done.OnDone(db.consumeFunc(ctx, req))
}

func newDisabledBatcher[T any](consumeFunc sender.SendFunc[T]) Batcher[T] {
	return &disabledBatcher[T]{consumeFunc: consumeFunc}
}
