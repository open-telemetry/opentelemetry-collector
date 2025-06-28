// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiterhelper // import "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"

import (
	"context"

	"go.opentelemetry.io/collector/extension/extensionlimiter"
)

// Wrapper is a general-purpose interface for limiter consumers
// to limit resources with use of a callback.  This is the simplest
// form of rate limiting interface from a callers perspective.  If the
// caller is a pipeline component, consider using a consumer-oriented
// limiterhelper (e.g., limiterhelper.NewLimitedLogs) to simplify
// construction of this interface.
//
// A wrapped limiter is either a RateLimiter or ResourceLimiter
// interface. Wrappers can be constructed from either of the
// underlying limiters and their corresponding providers. Usually
// configmiddleware or limiterhelper is responsible for constructing
// the correct wrapper from these two kinds of limiter; users will use
// this interface consistently.
type Wrapper interface {
	// LimitCall applies the limiter and with the rate or resource
	// granted makes a scoped call, returning success or an error
	// from either the limiter or the enclosed callback.
	//
	// The `call` parameter must be non-nil.
	LimitCall(ctx context.Context, weight int, call func(ctx context.Context) error) error
}

// LimitCallFunc is a functional way to build Wrappers.
type LimitCallFunc func(context.Context, int, func(ctx context.Context) error) error

// LimitCall implements part of the Wrapper interface.
func (f LimitCallFunc) LimitCall(ctx context.Context, value int, call func(ctx context.Context) error) error {
	if f == nil {
		return call(ctx)
	}
	return f(ctx, value, call)
}

// wrapperImpl is a functional Wrapper object.  The zero state is a
// no-op.
type wrapperImpl struct {
	LimitCallFunc
}

var _ Wrapper = wrapperImpl{}

// NewWrapperImpl returns a functional implementation of
// Wrapper.  Use a nil argument for the no-op implementation.
func NewWrapperImpl(f LimitCallFunc) Wrapper {
	return wrapperImpl{
		LimitCallFunc: f,
	}
}

// ResourceToWrapperProvider constructs a
// WrapperProvider for a resource limiter extension.
func ResourceToWrapperProvider(rp extensionlimiter.ResourceLimiterProvider) WrapperProvider {
	return NewWrapperProviderImpl(
		func(key extensionlimiter.WeightKey, opts ...extensionlimiter.Option) (Wrapper, error) {
			lim, err := rp.GetResourceLimiter(key, opts...)
			if err != nil {
				return nil, err
			}
			if lim == nil {
				return NewWrapperImpl(nil), nil
			}
			blocking := NewBlockingResourceLimiter(lim)
			return NewWrapperImpl(
				func(ctx context.Context, value int, call func(context.Context) error) error {
					release, err := blocking.WaitFor(ctx, value)
					if err != nil {
						return err
					}
					defer release()
					return call(ctx)
				},
			), nil
		})
}

// RateToWrapperProvider constructs a WrapperProvider
// for a rate limiter extension.
func RateToWrapperProvider(rp extensionlimiter.RateLimiterProvider) WrapperProvider {
	return NewWrapperProviderImpl(
		func(key extensionlimiter.WeightKey, opts ...extensionlimiter.Option) (Wrapper, error) {
			lim, err := rp.GetRateLimiter(key, opts...)
			if err != nil {
				return nil, err
			}
			if lim == nil {
				return NewWrapperImpl(nil), nil
			}
			blocking := NewBlockingRateLimiter(lim)
			return NewWrapperImpl(
				func(ctx context.Context, value int, call func(context.Context) error) error {
					if err := blocking.WaitFor(ctx, value); err != nil {
						return err
					}
					return call(ctx)
				},
			), nil
		},
	)
}
