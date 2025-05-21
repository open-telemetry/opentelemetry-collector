// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiterhelper // import "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/extension/extensionlimiter"
	"go.uber.org/multierr"
)

var (
	ErrNotALimiter       = errors.New("middleware is not a limiter")
	ErrNotARateLimiter   = errors.New("middleware cannot implement rate limiter")
	ErrLimiterConflict   = errors.New("limiter implements both rate and resource-limiters")
	ErrUnresolvedLimiter = errors.New("could not resolve middleware limiter")
)

// MiddlewareIsLimiter applies consistency checks and returns a valid
// limiter extension of any known kind.
func MiddlewareIsLimiter(host component.Host, middleware configmiddleware.Config) (extensionlimiter.BaseLimiterProvider, bool, error) {
	exts := host.GetExtensions()
	ext := exts[middleware.ID]
	if ext == nil {
		return nil, false, fmt.Errorf("%w: %s", ErrUnresolvedLimiter, middleware.ID)
	}
	_, isResource := ext.(extensionlimiter.ResourceLimiterProvider)
	_, isRate := ext.(extensionlimiter.RateLimiterProvider)
	base, isBase := ext.(extensionlimiter.BaseLimiterProvider)

	switch {
	case isResource && isRate:
		return nil, false, fmt.Errorf("%w: %s", ErrLimiterConflict, middleware.ID)
	case isResource, isRate:
		return base, true, nil
	case isBase:
		return base, true, nil
	default:
		return nil, false, nil
	}
}

// MiddlewaresToLimiterProvider constructs a combined limiter
// from an ordered list of middlewares. This constructor ignores
// middleware configs that are not limiters.
func MiddlewaresToLimiterProvider(host component.Host, middleware []configmiddleware.Config) (MultiLimiterProvider, error) {
	var retErr error
	var providers MultiLimiterProvider
	for _, mid := range middleware {
		base, ok, err := MiddlewareIsLimiter(host, mid)
		retErr = multierr.Append(retErr, err)
		if !ok {
			continue
		}
		providers = append(providers, base)
		retErr = multierr.Append(retErr, err)
	}
	if len(providers) == 0 {
		return nil, nil
	}
	return providers, nil
}

// MiddlewareToBaseLimiterProvider returns a base limiter provider
// from middleware. Returns a package-level error if the middleware
// does not implement exactly one of the limiter interfaces (i.e.,
// rate or resource).
func MiddlewareToBaseLimiterProvider(host component.Host, middleware configmiddleware.Config) (extensionlimiter.BaseLimiterProvider, error) {
	return getMiddleware(
		host,
		middleware,
		identity[extensionlimiter.BaseLimiterProvider],
		baseProvider[extensionlimiter.RateLimiterProvider],
		baseProvider[extensionlimiter.ResourceLimiterProvider],
	)
}

// MiddlewareToLimiterWrapperProvider returns a limiter wrapper
// provider from middleware. Returns a package-level error if the
// middleware does not implement exactly one of the limiter
// interfaces (i.e., rate or resource).
func MiddlewareToLimiterWrapperProvider(host component.Host, middleware configmiddleware.Config) (LimiterWrapperProvider, error) {
	return getMiddleware(
		host,
		middleware,
		nilError(BaseToLimiterWrapperProvider),
		nilError(RateToLimiterWrapperProvider),
		nilError(ResourceToLimiterWrapperProvider),
	)
}

// MiddlewareToRateLimiterProvider allows a base limiter provider to
// act as a rate limiter provider. This encodes the fact that a
// resource limiter extension cannot be adapted to a rate limiter
// interface. Returns a package-level error if the middleware does not
// implement exactly one of the limiter interfaces (i.e., rate or
// resource).
func MiddlewareToRateLimiterProvider(host component.Host, middleware configmiddleware.Config) (extensionlimiter.RateLimiterProvider, error) {
	return getMiddleware(
		host,
		middleware,
		nilError(BaseToRateLimiterProvider),
		identity[extensionlimiter.RateLimiterProvider],
		resourceToRateLimiterError,
	)
}

// MiddlewareToResourceLimiterProvider allows a base limiter provider
// to act as a resource provider. This enforces that a resource
// limiter extension cannot be adapted to a resource limiter
// interface. Returns a package-level error if the middleware does not
// implement exactly one of the limiter interfaces (i.e., rate or
// resource).
func MiddlewareToResourceLimiterProvider(host component.Host, middleware configmiddleware.Config) (extensionlimiter.ResourceLimiterProvider, error) {
	return getMiddleware(
		host,
		middleware,
		nilError(BaseToResourceLimiterProvider),
		nilError(RateToResourceLimiterProvider),
		identity[extensionlimiter.ResourceLimiterProvider],
	)
}

// getProvider invokes getProvider if any kind of limiter is detected
// for the given host and middleware configuration.
func getMiddleware[Out any](
	host component.Host,
	middleware configmiddleware.Config,
	base func(extensionlimiter.BaseLimiterProvider) (Out, error),
	rate func(extensionlimiter.RateLimiterProvider) (Out, error),
	resource func(extensionlimiter.ResourceLimiterProvider) (Out, error),
) (Out, error) {
	var out Out
	ext, ok, err := MiddlewareIsLimiter(host, middleware)
	if err != nil {
		return out, err
	}
	if !ok {
		return out, fmt.Errorf("%w: %s", ErrNotALimiter, middleware.ID)
	}
	return getProvider(ext, base, rate, resource)
}

// getProvider handles each limiter kind, case-by-case, for building
// limiters in a functional style.
func getProvider[Out any](
	ext extensionlimiter.BaseLimiterProvider,
	base func(extensionlimiter.BaseLimiterProvider) (Out, error),
	rate func(extensionlimiter.RateLimiterProvider) (Out, error),
	resource func(extensionlimiter.ResourceLimiterProvider) (Out, error),
) (Out, error) {
	if lim, ok := ext.(extensionlimiter.ResourceLimiterProvider); ok {
		return resource(lim)
	}
	if lim, ok := ext.(extensionlimiter.RateLimiterProvider); ok {
		return rate(lim)
	}
	if lim, ok := ext.(extensionlimiter.BaseLimiterProvider); ok {
		return base(lim)
	}
	var out Out
	return out, ErrNotALimiter
}

// identity is a pass-through for the correct provider type.
func identity[T any](lim T) (T, error) {
	return lim, nil
}

// baseProvider returns a base limiter type from any limiter.
func baseProvider[T extensionlimiter.BaseLimiterProvider](p T) (extensionlimiter.BaseLimiterProvider, error) {
	return p, nil
}

// nilError converts an infallible constructor to return a nil error.
func nilError[S, T any](f func(S) T) func(S) (T, error) {
	return func(s S) (T, error) { return f(s), nil }
}

// resourceToRateLimiterError represents the impossible conversion
// from resource limiter to rate limiter.
func resourceToRateLimiterError(_ extensionlimiter.ResourceLimiterProvider) (extensionlimiter.RateLimiterProvider, error) {
	return nil, ErrNotARateLimiter
}
