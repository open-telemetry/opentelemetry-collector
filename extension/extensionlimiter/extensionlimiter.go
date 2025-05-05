// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

import (
	"context"
)

// Defines:
// - Option
// - Checker
// - CheckerFunc

// Option is passed to limiter providers.
//
// NOTE: For data-specific or tenant-specific limits we will extend
// providers with Options and add a Config type, but none are
// supported yet and this PR contains only interfaces, not need for
// options in core repository components.
type Option interface {
	apply()
}

// Checker is for checking when a limit is saturated.  This can be
// called prior to the start of work to check for limiter saturation.
type Checker interface {
	// MustDeny is the logical equivalent of Acquire(0).  If the
	// Acquire would fail even for 0 units of a rate, the
	// caller must deny the request.  Implementations are
	// encouraged to ensure that when MustDeny() is false,
	// Acquire(0) is also false, however callers could use a
	// faster code path to implement MustDeny() since it does not
	// depend on the value.
	MustDeny(context.Context) error
}

// CheckerFunc is a functional way to build Checker implementations.
type CheckerFunc func(context.Context) error

var _ Checker = CheckerFunc(nil)

// MustDeny implements Checker.
func (f CheckerFunc) MustDeny(ctx context.Context) error {
	if f == nil {
		return nil
	}
	return f(ctx)
}

// PassThroughChecker returns a Checker that never denies.
func PassThroughChecker() Checker {
	return CheckerFunc(nil)
}
