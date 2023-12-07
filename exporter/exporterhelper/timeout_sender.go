// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"time"
)

// TimeoutSettings for timeout. The timeout applies to individual attempts to send data to the backend.
type TimeoutSettings struct {
	// Timeout is the timeout for every attempt to send data to the backend.
	Timeout time.Duration `mapstructure:"timeout"`
}

// NewDefaultTimeoutSettings returns the default settings for TimeoutSettings.
func NewDefaultTimeoutSettings() TimeoutSettings {
	return TimeoutSettings{
		Timeout: 5 * time.Second,
	}
}

// timeoutSender is a requestSender that adds a `timeout` to every request that passes this sender.
type timeoutSender struct {
	baseRequestSender
	cfg TimeoutSettings
}

func (ts *timeoutSender) send(ctx context.Context, req Request) error {
	// Intentionally don't overwrite the context inside the request, because in case of retries deadline will not be
	// updated because this deadline most likely is before the next one.
	if ts.cfg.Timeout > 0 {
		var cancelFunc func()
		ctx, cancelFunc = context.WithTimeout(ctx, ts.cfg.Timeout)
		defer cancelFunc()
	}
	return req.Export(ctx)
}
