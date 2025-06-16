// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiterhelper // import "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"

import (
	"errors"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/extension/extensionlimiter"
)

var (
	ErrNotALimiter     = errors.New("middleware is not a limiter")
	ErrNotARateLimiter = errors.New("middleware cannot implement rate limiter")
	ErrLimiterConflict = errors.New("limiter implements both rate and resource-limiters")
)

// middlewareCheck applies consistency checks and returns a valid
// limiter extension of any known kind.
func middlewareCheck(ext extensionlimiter.SaturationCheckerProvider) (extensionlimiter.SaturationCheckerProvider, error) {
	_, isResource := ext.(extensionlimiter.ResourceLimiterProvider)
	_, isRate := ext.(extensionlimiter.RateLimiterProvider)

	if isResource && isRate {
		return nil, ErrLimiterConflict
	}
	return ext, nil
}

// MultipleProvider constructs a combined limiter from an ordered list
// of middlewares. This constructor ignores middleware configs that
// are not limiters.
func MultipleProvider(exts []extensionlimiter.SaturationCheckerProvider) (MultiLimiterProvider, error) {
	var retErr error
	var providers MultiLimiterProvider
	for _, ext := range exts {
		base, err := middlewareCheck(ext)
		retErr = multierr.Append(retErr, err)
		providers = append(providers, base)
	}
	return providers, retErr
}

// MiddlewareToSaturationCheckerProvider returns a base limiter provider
// from middleware. Returns a package-level error if the middleware
// does not implement exactly one of the limiter interfaces (i.e.,
// rate or resource).
func MiddlewareToSaturationCheckerProvider(ext extensionlimiter.SaturationCheckerProvider) (extensionlimiter.SaturationCheckerProvider, error) {
	return getMiddleware(
		ext,
		identity[extensionlimiter.SaturationCheckerProvider],
		baseProvider[extensionlimiter.RateLimiterProvider],
		baseProvider[extensionlimiter.ResourceLimiterProvider],
	)
}

// MiddlewareToLimiterWrapperProvider returns a limiter wrapper
// provider from middleware. Returns a package-level error if the
// middleware does not implement exactly one of the limiter
// interfaces (i.e., rate or resource).
func MiddlewareToLimiterWrapperProvider(ext extensionlimiter.SaturationCheckerProvider) (LimiterWrapperProvider, error) {
	return getMiddleware(
		ext,
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
func MiddlewareToRateLimiterProvider(ext extensionlimiter.SaturationCheckerProvider) (extensionlimiter.RateLimiterProvider, error) {
	return getMiddleware(
		ext,
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
func MiddlewareToResourceLimiterProvider(ext extensionlimiter.SaturationCheckerProvider) (extensionlimiter.ResourceLimiterProvider, error) {
	return getMiddleware(
		ext,
		nilError(BaseToResourceLimiterProvider),
		nilError(RateToResourceLimiterProvider),
		identity[extensionlimiter.ResourceLimiterProvider],
	)
}

// getProvider invokes getProvider if any kind of limiter is detected
// for the given host and middleware configuration.
func getMiddleware[Out any](
	ext extensionlimiter.SaturationCheckerProvider,
	base func(extensionlimiter.SaturationCheckerProvider) (Out, error),
	rate func(extensionlimiter.RateLimiterProvider) (Out, error),
	resource func(extensionlimiter.ResourceLimiterProvider) (Out, error),
) (Out, error) {
	var out Out
	ext, err := middlewareCheck(ext)
	if err != nil {
		return out, err
	}
	return getProvider(ext, base, rate, resource)
}

// getProvider handles each limiter kind, case-by-case, for building
// limiters in a functional style.
func getProvider[Out any](
	ext extensionlimiter.SaturationCheckerProvider,
	base func(extensionlimiter.SaturationCheckerProvider) (Out, error),
	rate func(extensionlimiter.RateLimiterProvider) (Out, error),
	resource func(extensionlimiter.ResourceLimiterProvider) (Out, error),
) (Out, error) {
	if lim, ok := ext.(extensionlimiter.ResourceLimiterProvider); ok {
		return resource(lim)
	}
	if lim, ok := ext.(extensionlimiter.RateLimiterProvider); ok {
		return rate(lim)
	}
	if lim, ok := ext.(extensionlimiter.SaturationCheckerProvider); ok {
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
func baseProvider[T extensionlimiter.SaturationCheckerProvider](p T) (extensionlimiter.SaturationCheckerProvider, error) {
	return p, nil
}

// nilError converts an infallible constructor to return a nil error.
func nilError[S, T any](f func(S) T) func(S) (T, error) {
	return func(s S) (T, error) { return f(s), nil }
}
