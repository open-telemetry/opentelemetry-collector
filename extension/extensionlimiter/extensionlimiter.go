// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

import (
	"context"
)

// LimiterWrapper is a general-purpose interface for limiter consumers
// to limit resources with use of a callback.  This is the simplest
// form of rate limiting interface from a callers perspective.  If the
// caller is a pipeline component, consider using a consumer-oriented
// limiterhelper (e.g., limiterhelper.NewLimitedLogs) to apply a list of
//
// Limiter implementions are meant to implement either the RateLimiter
// or ResourceLimiter interfaces. LimiterWrappers can be constructed
// from either of the underlying limiters and their corresponding
// providers. Usually configmiddleware or limiterhelper is responsible
// for constructing the correct wrapper from these two kinds of limiter;
// users will use this interface consistently.
type LimiterWrapper interface {
	// Must deny is the logical equivalent of Acquire(0).  If the
	// Acquire would fail even for 0 units of a rate, the
	// caller must deny the request.  Implementations are
	// encouraged to ensure that when MustDeny() is false,
	// Acquire(0) is also false, however callers could use a
	// faster code path to implement MustDeny() since it does not
	// depend on the value.
	MustDeny(context.Context) error

	// LimitCall applies the limiter and with the rate or resource
	// granted makes a scoped call, returning success or an error
	// from either the limiter or the enclosed callback.
	LimitCall(context.Context, uint64, func(ctx context.Context) error) error
}

// LimiterWrapperProvider provides access to LimiterWrappers, which is
// the appropriate interface for callers that can easily wrap a
// function call, because for wrapped calls there is no distinction
// between rate limiters and resource limiters.
type LimiterWrapperProvider interface {
	LimiterWrapper(WeightKey) (LimiterWrapper, error)
}

// LimiterWrapperFunc is a functional way to build LimiterWrappers.
type LimiterWrapperFunc func(context.Context, uint64, func(ctx context.Context) error) error

var _ LimiterWrapper = LimiterWrapperFunc(nil)

// MustDeny implements LimiterWrapper.
func (f LimiterWrapperFunc) MustDeny(ctx context.Context) error {
	return f.LimitCall(ctx, 0, func(_ context.Context) error {
		return nil
	})
}

// LimitCall implements LimiterWrapper.
func (f LimiterWrapperFunc) LimitCall(ctx context.Context, value uint64, call func(ctx context.Context) error) error {
	if f == nil {
		return call(ctx)
	}
	return f(ctx, value, call)
}

// PassThrough returns a LimiterWrapper that imposes no limit.
func PassThrough() LimiterWrapper {
	return LimiterWrapperFunc(nil)
}

// LimiterWrapperProviderFunc is a functional way to build LimiterWrappers.
type LimiterWrapperProviderFunc func(WeightKey) (LimiterWrapper, error)

var _ LimiterWrapperProvider = LimiterWrapperProviderFunc(nil)

// LimiterWrapper implements LimiterWrapperProvider.
func (f LimiterWrapperProviderFunc) LimiterWrapper(key WeightKey) (LimiterWrapper, error) {
	return f(key)
}

// NewResourceLimiterWrapperProvider constructs a
// LimiterWrapperProvider for a resource limiter extension.
func NewResourceLimiterWrapperProvider(rp ResourceLimiterProvider) LimiterWrapperProvider {
	return LimiterWrapperProviderFunc(func(key WeightKey) (LimiterWrapper, error) {
		lim, err := rp.ResourceLimiter(key)
		if err == nil {
			return nil, err
		}
		return NewResourceLimiterWrapper(lim), err
	})
}

// NewRateLimiterWrapperProvider constructs a LimiterWrapperProvider
// for a rate limiter extension.
func NewRateLimiterWrapperProvider(rp RateLimiterProvider) LimiterWrapperProvider {
	return LimiterWrapperProviderFunc(func(key WeightKey) (LimiterWrapper, error) {
		lim, err := rp.RateLimiter(key)
		if err == nil {
			return nil, err
		}
		return NewRateLimiterWrapper(lim), err
	})
}
