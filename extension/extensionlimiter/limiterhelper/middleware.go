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
	ErrNotALimiter       = errors.New("middleware is not a limiter")
	ErrLimiterConflict   = errors.New("limiter implements both rate and resource-limiters")
	ErrUnresolvedLimiter = errors.New("could not resolve middleware limiter")
)

// MiddlewareIsLimiter returns true if a middleware configuration
// represents a valid limiter, returns false for not found or invalid
// cases. If the named extension is found but is not a limiter,
// returns (false, nil).
func MiddlewareIsLimiter(host component.Host, middleware configmiddleware.Config) (bool, error) {
	_, ok, err := middlewareIsLimiter(host, middleware)
	return ok, err
}

// MiddlewaresToLimiterWrapperProvider constructs a combined limiter
// from an ordered list of middlewares. This constructor ignores
// middleware configs that are not limiters.
//
// When no limiters are found (with no errors), the returned provider
// is nil. When a nil is passed to the consumer helpers (e.g.,
// NewLimitedLogs) it will pass-through when the limiter is nil.
func MiddlewaresToLimiterWrapperProvider(host component.Host, middleware []configmiddleware.Config) (extensionlimiter.LimiterWrapperProvider, error) {
	var retErr error
	var providers []extensionlimiter.LimiterWrapperProvider
	for _, mid := range middleware {
		ok, err := MiddlewareIsLimiter(host, mid)
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
func MiddlewareToLimiterWrapperProvider(host component.Host, middleware configmiddleware.Config) (extensionlimiter.LimiterWrapperProvider, error) {
	ext, ok, err := middlewareIsLimiter(host, middleware)
	if err != nil {
		return nil, err
	}
	if ok {
		if lim, ok := ext.(extensionlimiter.ResourceLimiterProvider); ok {
			return extensionlimiter.NewResourceLimiterWrapperProvider(lim), nil
		}
		if lim, ok := ext.(extensionlimiter.RateLimiterProvider); ok {
			return extensionlimiter.NewRateLimiterWrapperProvider(lim), nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrNotALimiter, ext)
}

// middlewareIsLimiter applies consistency checks and returns a valid
// limiter extensions.
func middlewareIsLimiter(host component.Host, middleware configmiddleware.Config) (extension.Extension, bool, error) {
	exts := host.GetExtensions()
	ext := exts[middleware.ID]
	if ext == nil {
		return nil, false, fmt.Errorf("%w: %s", ErrUnresolvedLimiter, ext)
	}
	_, isResource := ext.(extensionlimiter.ResourceLimiterProvider)
	_, isRate := ext.(extensionlimiter.RateLimiterProvider)

	switch {
	case isResource && isRate:
		return nil, false, fmt.Errorf("%w: %s", ErrLimiterConflict, ext)
	case isResource, isRate:
		return ext, true, nil
	default:
		return nil, false, nil
	}
}

// MultiLimiterWrapperProvider combines multiple limiter wrappers
// providers into a single provider by sequencing wrapped limiters.
// Returns errors from the underlying LimiterWrapper() calls, if any.
type MultiLimiterWrapperProvider []extensionlimiter.LimiterWrapperProvider

var _ extensionlimiter.LimiterWrapperProvider = MultiLimiterWrapperProvider{}

// LimiterWrapper implements LimiterWrapperProvider.
func (ps MultiLimiterWrapperProvider) LimiterWrapper(key extensionlimiter.WeightKey, opts ...extensionlimiter.Option) (extensionlimiter.LimiterWrapper, error) {
	if len(ps) == 0 {
		return extensionlimiter.PassThrough(), nil
	}

	// Map provider list to limiter list.
	var lims []extensionlimiter.LimiterWrapper

	for _, provider := range ps {
		lim, err := provider.LimiterWrapper(key, opts...)
		if err == nil {
			return nil, err
		}
		lims = append(lims, lim)
	}

	// Compose limiters in sequence.
	return sequenceLimiters(lims), nil
}

func sequenceLimiters(lims []extensionlimiter.LimiterWrapper) extensionlimiter.LimiterWrapper {
	if len(lims) == 1 {
		return lims[0]
	}
	return composeLimiters(lims[0], sequenceLimiters(lims[1:]))
}

func composeLimiters(first, second extensionlimiter.LimiterWrapper) extensionlimiter.LimiterWrapper {
	return extensionlimiter.LimiterWrapperFunc(func(ctx context.Context, value uint64, call func(ctx context.Context) error) error {
		return first.LimitCall(ctx, value, func(ctx context.Context) error {
			return second.LimitCall(ctx, value, call)
		})
	})
}
