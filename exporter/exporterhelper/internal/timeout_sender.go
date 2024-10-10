// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/config/configtimeout"
	"go.opentelemetry.io/collector/exporter/internal"
)

// TimeoutSender is a requestSender that adds a `timeout` to every request that passes this sender.
type TimeoutSender struct {
	BaseRequestSender
	cfg configtimeout.TimeoutConfig
}

func (ts *TimeoutSender) Send(ctx context.Context, req internal.Request) error {
	// If there is deadline that will expire before the configured timeout,
	// take one of several optional behaviors.
	if deadline, has := ctx.Deadline(); has {
		switch ts.cfg.ShortTimeoutPolicy {
		case configtimeout.PolicyIgnore:
			// The following call erases the
			// deadline, a new one will be
			// inserted below.
			ctx = context.WithoutCancel(ctx)
		case configtimeout.PolicyAbort:
			// We should return a gRPC-or-HTTP
			// response status saying the request
			// was aborted.
			if available := time.Until(deadline); available < ts.cfg.Timeout {
				return fmt.Errorf("request will be cancelled in %v before configured %v timeout", available, ts.cfg.Timeout)
			}
		case configtimeout.PolicySustain:
			// Allow the shorter-than-configured
			// timeout, means do nothing.
		}
	}
	// Note: testing for zero timeout happens after the deadline check
	// above, especially in case the user configures an "ignore" policy.
	if ts.cfg.Timeout == 0 {
		return req.Export(ctx)
	}
	// Intentionally don't overwrite the context inside the request, because in case of retries deadline will not be
	// updated because this deadline most likely is before the next one.
	tCtx, cancelFunc := context.WithTimeout(ctx, ts.cfg.Timeout)
	defer cancelFunc()
	return req.Export(tCtx)
}
