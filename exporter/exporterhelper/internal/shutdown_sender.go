// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"sync/atomic"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/experr"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

// shutdownSender marks the errors returned by the downstream senders as shutdown errors once the
// exporter starts shutting down. The persistent queue keeps the requests that fail with a shutdown
// error in the storage, so they are picked up again after a restart instead of being dropped.
//
// The retry sender reports a shutdown error only when a backoff wait is interrupted, which never
// happens for exporters that don't enable retry_on_failure. This sender is placed above the retry
// sender, so the requests that are in flight when the collector goes down are retained regardless
// of the retry configuration.
type shutdownSender[T any] struct {
	component.StartFunc
	shuttingDown atomic.Bool
	next         sender.Sender[T]
}

func newShutdownSender[T any](next sender.Sender[T]) *shutdownSender[T] {
	return &shutdownSender[T]{next: next}
}

func (ss *shutdownSender[T]) Shutdown(context.Context) error {
	ss.shuttingDown.Store(true)
	return nil
}

func (ss *shutdownSender[T]) Send(ctx context.Context, req T) error {
	err := ss.next.Send(ctx, req)
	// Permanent errors are not marked as shutdown errors, otherwise the persistent queue would keep
	// retaining data that always fails to be exported.
	if err == nil || consumererror.IsPermanent(err) || !ss.shuttingDown.Load() {
		return err
	}
	return experr.NewShutdownErr(err)
}
