// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiterhelper // import "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionlimiter"
)

var (
	ErrNotALimiter         = errors.New("middleware is not a limiter")
	ErrNotARateLimiter     = errors.New("middleware is not a rate or base limiter")
	ErrNotAResourceLimiter = errors.New("middleware is not a resource or base limiter")
	ErrLimiterConflict     = errors.New("limiter implements both rate and resource-limiters")
	ErrUnresolvedLimiter   = errors.New("could not resolve middleware limiter")
)

// MiddlewaresToLimiterWrapperProvider constructs a combined limiter
// from an ordered list of middlewares. This constructor ignores
// middleware configs that are not limiters.
//
// When no limiters are found (with no errors), the returned provider
// is nil. When a nil is passed to the consumer helpers (e.g.,
// NewLimitedLogs) it will pass-through when the limiter is nil.
func MiddlewaresToLimiterWrapperProvider(host component.Host, middleware []configmiddleware.Config) (LimiterWrapperProvider, error) {
	var retErr error
	var providers []LimiterWrapperProvider
	for _, mid := range middleware {
		_, ok, err := MiddlewareIsLimiter(host, mid)
		retErr = errors.Join(retErr, err)
		if !ok {
			continue
		}
		provider, err := MiddlewareToLimiterWrapperProvider(host, mid)
		providers = append(providers, provider)
		retErr = errors.Join(retErr, err)
	}
	if len(providers) == 0 {
		return nil, nil
	}
	return MultiLimiterWrapperProvider(providers), nil
}

// Note: MiddlewaresToRateLimiterProvider, MiddlewaresToResourceLimiterProvider
// are needed for special cases, however these functions can be implemented
// manually, they are similar to the above.

// MiddlewareToLimiterWrapperProvider returns a limiter wrapper
// provider from middleware. Returns a package-level error if the
// middleware does not implement exactly one of the limiter
// interfaces (i.e., rate or resource).
func MiddlewareToLimiterWrapperProvider(host component.Host, middleware configmiddleware.Config) (LimiterWrapperProvider, error) {
	ext, ok, err := MiddlewareIsLimiter(host, middleware)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotALimiter, middleware.ID)
	}
	if lim, ok := ext.(extensionlimiter.ResourceLimiterProvider); ok {
		return NewResourceLimiterWrapperProvider(lim), nil
	}
	if lim, ok := ext.(extensionlimiter.RateLimiterProvider); ok {
		return NewRateLimiterWrapperProvider(lim), nil
	}
	if lim, ok := ext.(extensionlimiter.BaseLimiterProvider); ok {
		return NewBaseLimiterWrapperProvider(lim), nil
	}
	// This is an internal error.
	return nil, fmt.Errorf("%w: %s: unrecognized limiter", ErrNotALimiter, middleware.ID)
}

// MiddlewareIsLimiter applies consistency checks and returns a valid
// limiter extension of any known kind.
func MiddlewareIsLimiter(host component.Host, middleware configmiddleware.Config) (extension.Extension, bool, error) {
	exts := host.GetExtensions()
	ext := exts[middleware.ID]
	if ext == nil {
		return nil, false, fmt.Errorf("%w: %s", ErrUnresolvedLimiter, middleware.ID)
	}
	_, isResource := ext.(extensionlimiter.ResourceLimiterProvider)
	_, isRate := ext.(extensionlimiter.RateLimiterProvider)
	_, isBase := ext.(extensionlimiter.BaseLimiterProvider)

	switch {
	case isResource && isRate:
		return ext, false, fmt.Errorf("%w: %s", ErrLimiterConflict, middleware.ID)
	case isResource, isRate:
		return ext, true, nil
	case isBase:
		return ext, true, nil
	default:
		return nil, false, nil
	}
}

// MiddlewareToRateLimiterProvider allows a base limiter to act as a
// rate. This encodes the fact that a resource limiter extension
// cannot be adapted to a rate limiter interface.
func MiddlewareToRateLimiterProvider(host component.Host, middleware configmiddleware.Config) (extensionlimiter.RateLimiterProvider, error) {
	ext, ok, err := MiddlewareIsLimiter(host, middleware)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotALimiter, middleware.ID)
	}
	if rlimp, ok := ext.(extensionlimiter.RateLimiterProvider); ok {
		return rlimp, nil
	}
	if blimp, ok := ext.(extensionlimiter.BaseLimiterProvider); ok {
		return BaseToRateLimiterProvider(blimp)
	}
	return nil, fmt.Errorf("%w: %s", ErrNotARateLimiter, middleware.ID)
}

// MultiLimiterWrapperProvider combines multiple limiter wrappers
// providers into a single provider by sequencing wrapped limiters.
// Returns errors from the underlying LimiterWrapper() calls, if any.
type MultiLimiterWrapperProvider []LimiterWrapperProvider

var _ LimiterWrapperProvider = MultiLimiterWrapperProvider{}

// GetLimiterWrapper implements LimiterWrapperProvider, combining
// checkers for all wrappers in a sequence.
func (ps MultiLimiterWrapperProvider) GetBaseLimiter(opts ...extensionlimiter.Option) (extensionlimiter.BaseLimiter, error) {
	var retErr error
	var cks MultiBaseLimiter
	for _, provider := range ps {
		ck, err := provider.GetBaseLimiter(opts...)
		retErr = errors.Join(retErr, err)
		if ck == nil {
			continue
		}
		cks = append(cks, ck)
	}
	if len(cks) == 0 {
		return extensionlimiter.MustDenyFunc(nil), retErr
	}
	return cks, retErr
}

// GetLimiterWrapper implements LimiterWrapperProvider, calling
// wrappers in a sequence.
func (ps MultiLimiterWrapperProvider) GetLimiterWrapper(key extensionlimiter.WeightKey, opts ...extensionlimiter.Option) (LimiterWrapper, error) {
	// Map provider list to limiter list.
	var lims []LimiterWrapper

	for _, provider := range ps {
		lim, err := provider.GetLimiterWrapper(key, opts...)
		if err == nil {
			return nil, err
		}
		if lim == nil {
			continue
		}
		lims = append(lims, lim)
	}

	if len(lims) == 0 {
		return LimiterWrapper(nil), nil
	}

	return sequenceLimiters(lims), nil
}

func sequenceLimiters(lims []LimiterWrapper) LimiterWrapper {
	if len(lims) == 1 {
		return lims[0]
	}
	return composeLimiters(lims[0], sequenceLimiters(lims[1:]))
}

func composeLimiters(first, second LimiterWrapper) LimiterWrapper {
	return LimiterWrapperFunc(func(ctx context.Context, value int, call func(ctx context.Context) error) error {
		return first.LimitCall(ctx, value, func(ctx context.Context) error {
			return second.LimitCall(ctx, value, call)
		})
	})
}
