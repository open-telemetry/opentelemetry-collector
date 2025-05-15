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

// BaseLimiter is for checking when a limit is saturated.  This can be
// called prior to the start of work to check for limiter saturation.
type BaseLimiter interface {
	// MustDeny is a request to apply a hard limit. If this
	// returns non-nil, the caller must not begin new work in this
	// context.
	MustDeny(context.Context) error
}

// MustDenyFunc is a functional way to build MustDeny functions.
type MustDenyFunc func(context.Context) error

// A MustDeny function is a complete BaseLimiter.
var _ BaseLimiter = MustDenyFunc(nil)

// MustDeny implements BaseLimiter.
func (f MustDenyFunc) MustDeny(ctx context.Context) error {
	if f == nil {
		return nil
	}
	return f(ctx)
}

// BaseLimiterProvider is an interface to obtain checkers for a group of
// weight keys.
type BaseLimiterProvider interface {
	// GetBaseLimiter returns a checker for a group of weight keys.
	GetBaseLimiter(...Option) (BaseLimiter, error)
}

// GetBaseLimiterFunc is a functional way to construct GetBaseLimiter
// functions, used in limiter providers.
type GetBaseLimiterFunc func(...Option) (BaseLimiter, error)

// BaseLimiter implements BaseLimiterProvider.
func (f GetBaseLimiterFunc) GetBaseLimiter(opts ...Option) (BaseLimiter, error) {
	if f == nil {
		return nil, nil
	}
	return f(opts...)
}
