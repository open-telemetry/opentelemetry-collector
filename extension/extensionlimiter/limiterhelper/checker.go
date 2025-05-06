// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiterhelper // import "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"

import (
	"context"

	"go.opentelemetry.io/collector/extension/extensionlimiter"
)

// MultiChecker returns MustDeny when any element returns MustDeny.
type MultiChecker []extensionlimiter.Checker

var _ extensionlimiter.Checker = MultiChecker{}

// MustDeny implements Checker.
func (ls MultiChecker) MustDeny(ctx context.Context) error {
	for _, lim := range ls {
		if lim == nil {
			continue
		}
		if err := lim.MustDeny(ctx); err != nil {
			return err
		}
	}
	return nil
}

// NewLimiterWrapperChecker returns a Checker for a LimiterWrapper.
func NewLimiterWrapperChecker(limiter LimiterWrapper) extensionlimiter.Checker {
	return extensionlimiter.MustDenyFunc(func(ctx context.Context) error {
		return limiter.LimitCall(ctx, 0, func(_ context.Context) error { return nil })
	})
}

// NewRateLimiterChecker returns a Checker for a RateLimiter.
func NewRateLimiterChecker(limiter extensionlimiter.RateLimiter) extensionlimiter.Checker {
	return extensionlimiter.MustDenyFunc(func(ctx context.Context) error {
		return limiter.Limit(ctx, 0)
	})
}

// NewResourceLimiterChecker returns a Checker for ResourceLimiter.
func NewResourceLimiterChecker(limiter extensionlimiter.ResourceLimiter) extensionlimiter.Checker {
	return extensionlimiter.MustDenyFunc(func(ctx context.Context) error {
		_, err := limiter.Acquire(ctx, 0)
		return err
	})
}
