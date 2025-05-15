// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

import (
	"context"
)

// RateLimiterProvider is a provider for rate limiters.
//
// Limiter implementations will implement this or the
// ResourceLimiterProvider interface, but MUST not implement both.
// Limiters are covered by configmiddleware configuration, which is
// able to construct LimiterWrappers from these providers.
type RateLimiterProvider interface {
	BaseLimiterProvider

	// GetRateLimiter returns a rate limiter for a weight key.
	GetRateLimiter(WeightKey, ...Option) (RateLimiter, error)
}

// GetRateLimiterFunc is a functional way to construct GetRateLimiter
// functions.
type GetRateLimiterFunc func(WeightKey, ...Option) (RateLimiter, error)

// RateLimiter implements RateLimiterProvider.
func (f GetRateLimiterFunc) GetRateLimiter(key WeightKey, opts ...Option) (RateLimiter, error) {
	if f == nil {
		return nil, nil
	}
	return f(key, opts...)
}

var _ RateLimiterProvider = struct {
	GetRateLimiterFunc
	GetBaseLimiterFunc
}{}

// RateLimiter is an interface that an implementation makes available
// to apply time-based limits on quantities such as the number of
// bytes or items per second.
//
// This is a relatively low-level interface. Callers that can use a
// LimiterWrapper should choose that interface instead. This interface
// is meant for direct use only in special cases where control flow
// cannot be easily scoped to a callback, for example inside
// middleware (e.g., grpc.StatsHandler).
//
// See the README for more recommendations.
type RateLimiter interface {
	// Limit attempts to apply rate limiting with the provided
	// weight, based on the key that was given to the provider.
	//
	// This is expected to block the caller until the weight can
	// be admitted, or when the limit is completely saturated,
	// limiters may also return immediate errors.
	Limit(ctx context.Context, value uint64) error
}

// LimitFunc is a functional way to construct Limit functions.
type LimitFunc func(ctx context.Context, value uint64) error

// Limit implements part of the RateLimiter interface.
func (f LimitFunc) Limit(ctx context.Context, value uint64) error {
	if f == nil {
		return nil
	}
	return f(ctx, value)
}

var _ RateLimiter = LimitFunc(nil)
