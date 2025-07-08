// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiterhelper // import "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"

import (
	"errors"

	"go.opentelemetry.io/collector/extension/extensionlimiter"
)

var (
	ErrNotALimiter     = errors.New("middleware is not a limiter")
	ErrNotARateLimiter = errors.New("middleware cannot implement rate limiter")
	ErrLimiterConflict = errors.New("limiter implements both rate and resource-limiters")
)

// AnyToWrapperProvider returns a limiter wrapper provider from
// middleware. Returns a package-level error if the middleware does
// not implement exactly one of the limiter interfaces (i.e., rate or
// resource).
func AnyToWrapperProvider(ext extensionlimiter.AnyProvider) (WrapperProvider, error) {
	return getAny(
		ext,
		nilError(RateToWrapperProvider),
		nilError(ResourceToWrapperProvider),
	)
}

// AnyToRateLimiterProvider allows a base limiter provider to act as a
// rate limiter provider. This encodes the fact that a resource
// limiter extension cannot be adapted to a rate limiter
// interface. Returns a package-level error if the middleware does not
// implement exactly one of the limiter interfaces (i.e., rate or
// resource).
func AnyToRateLimiterProvider(ext extensionlimiter.AnyProvider) (extensionlimiter.RateLimiterProvider, error) {
	return getAny(
		ext,
		identity[extensionlimiter.RateLimiterProvider],
		resourceToRateLimiterError,
	)
}

// AnyToResourceLimiterProvider allows a base limiter provider to act
// as a resource provider. This enforces that a resource limiter
// extension cannot be adapted to a resource limiter
// interface. Returns a package-level error if the middleware does not
// implement exactly one of the limiter interfaces (i.e., rate or
// resource).
func AnyToResourceLimiterProvider(ext extensionlimiter.AnyProvider) (extensionlimiter.ResourceLimiterProvider, error) {
	return getAny(
		ext,
		nilError(RateToResourceLimiterProvider),
		identity[extensionlimiter.ResourceLimiterProvider],
	)
}

// getAny invokes getProvider if any kind of limiter is detected for
// the given host and middleware configuration.
func getAny[Out any](
	ext extensionlimiter.AnyProvider,
	rate func(extensionlimiter.RateLimiterProvider) (Out, error),
	resource func(extensionlimiter.ResourceLimiterProvider) (Out, error),
) (Out, error) {
	var out Out
	res, isResource := ext.(extensionlimiter.ResourceLimiterProvider)
	rat, isRate := ext.(extensionlimiter.RateLimiterProvider)
	if isResource && isRate {
		return out, ErrLimiterConflict
	}
	if isResource {
		return resource(res)
	}
	if isRate {
		return rate(rat)
	}
	return out, ErrNotALimiter
}

// identity is a pass-through for the matching provider type.
func identity[T any](lim T) (T, error) {
	return lim, nil
}

// nilError converts an infallible constructor to return a nil error.
func nilError[S, T any](f func(S) T) func(S) (T, error) {
	return func(s S) (T, error) {
		return f(s), nil
	}
}

// resourceToRateLimiterError represents the impossible conversion
// from resource limiter to rate limiter.
func resourceToRateLimiterError(_ extensionlimiter.ResourceLimiterProvider) (extensionlimiter.RateLimiterProvider, error) {
	return nil, ErrNotARateLimiter
}
