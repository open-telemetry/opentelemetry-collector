// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

import (
	"context"
)

// Option is passed to limiter providers.
//
// NOTE: For data-specific or tenant-specific limits we will extend
// providers with Options and add a Config type, but none are
// supported yet and this PR contains only interfaces, not need for
// options in core repository components.
type Option interface {
	apply()
}

// Limiter is the common functionality implemented by LimiterWrapper,
// RateLimiter, and ResourceLimiter. This can be called prior to the
// start of work to check for limiter saturation.
type Limiter interface {
	// Must deny is the logical equivalent of Acquire(0).  If the
	// Acquire would fail even for 0 units of a rate, the
	// caller must deny the request.  Implementations are
	// encouraged to ensure that when MustDeny() is false,
	// Acquire(0) is also false, however callers could use a
	// faster code path to implement MustDeny() since it does not
	// depend on the value.
	MustDeny(context.Context) error
}

// LimiterFunc is a functional way to build MustDeny functions.
type LimiterFunc func(context.Context) error

var _ Limiter = LimiterFunc(nil)

// MustDeny implements Limiter.
func (f LimiterFunc) MustDeny(ctx context.Context) error {
	if f == nil {
		return nil
	}
	return f(ctx)
}
