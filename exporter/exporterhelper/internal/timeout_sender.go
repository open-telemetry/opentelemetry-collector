// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

// timeoutSender is a requestSender that adds a `timeout` to every request that passes this sender.
type timeoutSender[T any] struct {
	component.StartFunc
	component.ShutdownFunc
	cfg  TimeoutConfig
	next sender.Sender[T]
}

func newTimeoutSender[T any](cfg TimeoutConfig, next sender.Sender[T]) sender.Sender[T] {
	return &timeoutSender[T]{cfg: cfg, next: next}
}

func (ts *timeoutSender[T]) Send(ctx context.Context, req T) error {
	// Intentionally don't overwrite the context inside the request, because in case of retries deadline will not be
	// updated because this deadline most likely is before the next one.
	tCtx, cancelFunc := context.WithTimeout(ctx, ts.cfg.Timeout)
	defer cancelFunc()
	return ts.next.Send(tCtx, req)
}
