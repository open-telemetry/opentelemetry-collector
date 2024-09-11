// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"
	"time"
)

// Deprecated: [v0.110.0] Use TimeoutConfig instead.
type TimeoutSettings = TimeoutConfig

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

// Deprecated: [v0.110.0] Use NewDefaultTimeoutConfig instead.
func NewDefaultTimeoutSettings() TimeoutSettings {
	return NewDefaultTimeoutConfig()
}

// NewDefaultTimeoutConfig returns the default config for TimeoutConfig.
func NewDefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Timeout: 5 * time.Second,
	}
}

// timeoutSender is a requestSender that adds a `timeout` to every request that passes this sender.
type timeoutSender struct {
	baseRequestSender
	cfg TimeoutConfig
}

func (ts *timeoutSender) send(ctx context.Context, req Request) error {
	// TODO: Remove this by avoiding to create the timeout sender if timeout is 0.
	if ts.cfg.Timeout == 0 {
		return req.Export(ctx)
	}
	// Intentionally don't overwrite the context inside the request, because in case of retries deadline will not be
	// updated because this deadline most likely is before the next one.
	tCtx, cancelFunc := context.WithTimeout(ctx, ts.cfg.Timeout)
	defer cancelFunc()
	return req.Export(tCtx)
}
