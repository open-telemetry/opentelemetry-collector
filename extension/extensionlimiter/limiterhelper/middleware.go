// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiterhelper // import "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/extension/extensionlimiter"
)

var (
	ErrNotALimiter         = errors.New("middleware is not a limiter")
	ErrNotARateLimiter     = errors.New("middleware is not a rate or base limiter")
	ErrNotAResourceLimiter = errors.New("middleware is not a resource or base limiter")
	ErrLimiterConflict     = errors.New("limiter implements both rate and resource-limiters")
	ErrUnresolvedLimiter   = errors.New("could not resolve middleware limiter")
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

// MiddlewaresToLimiterWrapperProvider constructs a combined limiter
// from an ordered list of middlewares. This constructor ignores
// middleware configs that are not limiters.
//
// When no limiters are found (with no errors), the returned provider
// is nil. When a nil is passed to the consumer helpers (e.g.,
// NewLimitedLogs) it will pass-through when the limiter is nil.
func MiddlewaresToLimiterWrapperProvider(host component.Host, middleware []configmiddleware.Config) (LimiterWrapperProvider, error) {
	var retErr error
	var providers []extensionlimiter.BaseLimiterProvider
	for _, mid := range middleware {
		base, ok, err := MiddlewareIsLimiter(host, mid)
		retErr = errors.Join(retErr, err)
		if !ok {
			continue
		}
		providers = append(providers, base)
		retErr = errors.Join(retErr, err)
	}
	if len(providers) == 0 {
		return nil, nil
	}
	return MultiLimiterProvider(providers), nil
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

// MiddlewareToResourceLimiterProvider allows a base limiter to act as a
// resource. This encodes the fact that a resource limiter extension
// cannot be adapted to a resource limiter interface.
func MiddlewareToResourceLimiterProvider(host component.Host, middleware configmiddleware.Config) (extensionlimiter.ResourceLimiterProvider, error) {
	ext, ok, err := MiddlewareIsLimiter(host, middleware)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotALimiter, middleware.ID)
	}
	return getProvider(ext,
		BaseToResourceLimiterProvider,
		RateToResourceLimiterProvider,
		identity[extensionlimiter.ResourceLimiterProvider],
	)
}

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

func identity[T any](lim T) (T, error) {
	return lim, nil
}

func nilError[T any](f func() T) (T, error) {
	return f(), nil
}
