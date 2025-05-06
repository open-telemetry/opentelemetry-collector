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

// Checker is for checking when a limit is saturated.  This can be
// called prior to the start of work to check for limiter saturation.
type Checker interface {
	// MustDeny is a request to apply a hard limit. If this
	// returns non-nil, the caller must not begin new work in this
	// context.
	MustDeny(context.Context) error
}

// MustDenyFunc is a functional way to build MustDeny functions.
type MustDenyFunc func(context.Context) error

// A MustDeny function is a complete Checker.
var _ Checker = MustDenyFunc(nil)

// MustDeny implements Checker.
func (f MustDenyFunc) MustDeny(ctx context.Context) error {
	if f == nil {
		return nil
	}
	return f(ctx)
}

// NeverDeny returns a Checker that never denies.
func NeverDeny() Checker {
	return MustDenyFunc(nil)
}
