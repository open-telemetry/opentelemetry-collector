// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

// TimeoutConfig for timeout. The timeout applies to individual attempts to send data to the backend.
type TimeoutConfig struct {
	// Timeout is the timeout for every attempt to send data to the backend.
	// A zero timeout means no timeout.
	Timeout time.Duration `mapstructure:"timeout"`
}

func (ts *TimeoutConfig) Validate() error {
	// Negative timeouts are not acceptable, since all sends will fail.
	if ts.Timeout < 0 {
		return errors.New("'timeout' must be non-negative")
	}
	return nil
}

// NewDefaultTimeoutConfig returns the default config for TimeoutConfig.
func NewDefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Timeout: 5 * time.Second,
	}
}

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
