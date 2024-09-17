// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Deprecated: [v0.110.0] Use TimeoutConfig instead.
type TimeoutSettings = TimeoutConfig

// TimeoutConfig for timeout. The timeout applies to individual attempts to send data to the backend.
type TimeoutConfig struct {
	// Timeout is the timeout for every attempt to send data to the backend.
	// A zero timeout means no timeout.
	Timeout time.Duration `mapstructure:"timeout"`

	// OnShortTimeout is the exporter's disposition toward short timeouts,
	// any time the incoming context's deadline is shorter than the
	// configured Timeout.  The choices are:
	//
	// - sustain: use the shorter-than-configured timeout
	// - abort: abort the request for having insufficient deadline on arrival
	// - ignore: ignore the previous deadline and proceed with the configured timeout
	OnShortTimeout string `mapstructure:"on_short_timeout"`
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
		Timeout:        5 * time.Second,
		OnShortTimeout: "sustain",
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
	// If there is deadline that will expire before the configured timeout,
	// take one of several optional behaviors.
	if deadline, has := ctx.Deadline(); has {
		available := time.Until(deadline)
		if available < ts.cfg.Timeout {
			switch ts.cfg.OnShortTimeout {
			case "ignore":
				// The following call erases the
				// deadline, a new one will be
				// inserted below.
				ctx = context.WithoutCancel(ctx)
			case "abort":
				// We should return a gRPC-or-HTTP
				// response status saying the request
				// was aborted.
				return fmt.Errorf("Aborted: context deadline %v shorter than configured timeout %v", available, ts.cfg.Timeout)
			case "sustain":
				// Allow the shorter-than-configured
				// timeout, means do nothing.
			default:
				// Default is sustain, the legacy behavior.
			}
		}
	}

	tCtx, cancelFunc := context.WithTimeout(ctx, ts.cfg.Timeout)
	defer cancelFunc()
	return req.Export(tCtx)
}
