// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"
	"time"
)

// TimeoutSettings for timeout. The timeout applies to individual attempts to send data to the backend.
type TimeoutSettings struct {
	// Timeout is the timeout for every attempt to send data to the backend.
	// A zero timeout means no timeout.
	Timeout time.Duration `mapstructure:"timeout"`
}

func (ts *TimeoutSettings) Validate() error {
	// Negative timeouts are not acceptable, since all sends will fail.
	if ts.Timeout < 0 {
		return errors.New("'timeout' must be non-negative")
	}
	return nil
}

// NewDefaultTimeoutSettings returns the default settings for TimeoutSettings.
func NewDefaultTimeoutSettings() TimeoutSettings {
	return TimeoutSettings{
		Timeout: 5 * time.Second,
	}
}

func newTimeoutSender(timeoutConfig TimeoutSettings, nextSender requestSender) requestSender {
	return &timeoutSender{baseRequestSender: baseRequestSender{nextSender: nextSender}, cfg: timeoutConfig}
}

// timeoutSender is a requestSender that adds a `timeout` to every request that passes this sender.
type timeoutSender struct {
	baseRequestSender
	cfg TimeoutSettings
}

func (ts *timeoutSender) send(ctx context.Context, req Request) error {
	// TODO: Remove this by avoiding to create the timeout sender if timeout is 0.
	if ts.cfg.Timeout == 0 {
		return req.Export(ctx)
	}
	tCtx, cancelFunc := context.WithTimeout(ctx, ts.cfg.Timeout)
	defer cancelFunc()
	return ts.nextSender.send(tCtx, req)
}
