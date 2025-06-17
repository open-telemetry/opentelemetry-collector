// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiterhelper // import "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"

import (
	"context"

	"go.opentelemetry.io/collector/extension/extensionlimiter"
)

// LimiterWrapperProvider follows the provider pattern for
// the LimiterWrapper type
type LimiterWrapperProvider interface {
	GetLimiterWrapper(extensionlimiter.WeightKey, ...extensionlimiter.Option) (LimiterWrapper, error)
}

// GetLimiterWrapperFunc is an easy way to build GetLimiterWrapper functions.
type GetLimiterWrapperFunc func(extensionlimiter.WeightKey, ...extensionlimiter.Option) (LimiterWrapper, error)

// GetLimiterWrapper implements LimiterWrapperProvider.
func (f GetLimiterWrapperFunc) GetLimiterWrapper(key extensionlimiter.WeightKey, opts ...extensionlimiter.Option) (LimiterWrapper, error) {
	if f == nil {
		return LimiterWrapperFunc(nil), nil
	}
	return f(key, opts...)
}

type limiterWrapperProvider struct {
	GetLimiterWrapperFunc
}

var _ LimiterWrapperProvider = limiterWrapperProvider{}

// LimiterWrapper is a general-purpose interface for limiter consumers
// to limit resources with use of a callback.  This is the simplest
// form of rate limiting interface from a callers perspective.  If the
// caller is a pipeline component, consider using a consumer-oriented
// limiterhelper (e.g., limiterhelper.NewLimitedLogs) to simplify
// construction of this interface.
//
// A wrapped limiter is either a RateLimiter or ResourceLimiter
// interface. LimiterWrappers can be constructed from either of the
// underlying limiters and their corresponding providers. Usually
// configmiddleware or limiterhelper is responsible for constructing
// the correct wrapper from these two kinds of limiter; users will use
// this interface consistently.
type LimiterWrapper interface {
	// LimitCall applies the limiter and with the rate or resource
	// granted makes a scoped call, returning success or an error
	// from either the limiter or the enclosed callback.
	//
	// The `call` parameter must be non-nil.
	LimitCall(ctx context.Context, weight int, call func(ctx context.Context) error) error
}

// LimiterWrapperFunc is a functional way to build LimiterWrappers.
type LimiterWrapperFunc func(context.Context, int, func(ctx context.Context) error) error

var _ LimiterWrapper = LimiterWrapperFunc(nil)

// LimitCall implements LimiterWrapper.
func (f LimiterWrapperFunc) LimitCall(ctx context.Context, value int, call func(ctx context.Context) error) error {
	if f == nil {
		return call(ctx)
	}
	return f(ctx, value, call)
}

// ResourceToLimiterWrapperProvider constructs a
// LimiterWrapperProvider for a resource limiter extension.
func ResourceToLimiterWrapperProvider(rp extensionlimiter.ResourceLimiterProvider) LimiterWrapperProvider {
	return limiterWrapperProvider{
		GetLimiterWrapperFunc: func(key extensionlimiter.WeightKey, opts ...extensionlimiter.Option) (LimiterWrapper, error) {
			lim, err := rp.GetResourceLimiter(key, opts...)
			if err != nil {
				return nil, err
			}
			if lim == nil {
				return nil, nil
			}
			blocking := NewBlockingResourceLimiter(lim)
			return LimiterWrapperFunc(func(ctx context.Context, value int, call func(context.Context) error) error {
				release, err := blocking.WaitFor(ctx, value)
				if err != nil {
					return err
				}
				defer release()
				return call(ctx)
			}), nil
		},
	}
}

// RateToLimiterWrapperProvider constructs a LimiterWrapperProvider
// for a rate limiter extension.
func RateToLimiterWrapperProvider(rp extensionlimiter.RateLimiterProvider) LimiterWrapperProvider {
	return limiterWrapperProvider{
		GetLimiterWrapperFunc: func(key extensionlimiter.WeightKey, opts ...extensionlimiter.Option) (LimiterWrapper, error) {
			lim, err := rp.GetRateLimiter(key, opts...)
			if err != nil {
				return nil, err
			}
			if lim == nil {
				return nil, nil
			}
			blocking := NewBlockingRateLimiter(lim)
			return LimiterWrapperFunc(func(ctx context.Context, value int, call func(context.Context) error) error {
				if err := blocking.WaitFor(ctx, value); err != nil {
					return err
				}
				return call(ctx)
			}), nil
		},
	}
}
