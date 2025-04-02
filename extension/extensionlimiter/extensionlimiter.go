// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

import (
	"context"
)

// WeightKey is an enum type for common rate limits
type WeightKey string

// Predefined weight keys for common rate limits.  This is not a closed set.
//
// Providers should return errors when they do not recognize a weight key.
const (
	// WeightKeyNetworkBytes is typically used with RateLimiters
	// for limiting arrival rate.
	WeightKeyNetworkBytes WeightKey = "network_bytes"

	// WeightKeyRequestCount can be used to limit the rate or
	// total concurrent number of requests (i.e., pipeline data
	// objects).
	WeightKeyRequestCount WeightKey = "request_count"

	// WeightKeyRequestItems can be used to limit the rate or
	// total concurrent number of items (log records, metric data
	// points, spans, profiles).
	WeightKeyRequestItems WeightKey = "request_items"

	// WeightKeyMemorySize is typically used with ResourceLimiters
	// for limiting active memory usage.
	WeightKeyMemorySize WeightKey = "memory_size"
)

// Provider is an interface that provides access to different limiter types
// for specific weight keys.
//
// Extensions implementing this interface can be referenced by their
// names from component rate limiting configurations (e.g., limitermiddleware).
type Provider interface {
	// RateLimiter returns a RateLimiter for the specified weight key.
	//
	// Rate limiters are useful for limiting an amount of
	// something per unit of time.  In the OpenTelemetry metrics
	// data model, resource limiters would apply to value of an
	// Counter, where the value is incremented on Limit().
	RateLimiter(key WeightKey) RateLimiter

	// ResourceLimiter returns a ResourceLimiter for the specified
	// weight key.
	//
	// Resource limiters are useful for limiting a total amount of
	// something.  In the OpenTelemetry metrics data model,
	// resource limiters would apply to value of an UpDownCounter,
	// where the value is incremented on Acquire() and decremented
	// in Release().
	//
	// In cases where a component supports a rate limiter and does not use
	// a release function, the component may return a ResourceLimiterFunc
	// which calls the underlying rate limiter and returns a no-op ReleaseFunc.
	ResourceLimiter(key WeightKey) ResourceLimiter
}

// ResourceLimiter is an interface that components can use to apply
// resource limiting (e.g., concurrent requests, memory in use).
type ResourceLimiter interface {
	// Acquire attempts to acquire resources based on the provided weight value.
	//
	// It may block until resources are available or return an error if the limit
	// cannot be satisfied.
	//
	// On success, it returns a ReleaseFunc that should be called
	// when the resources are no longer needed.
	Acquire(ctx context.Context, value uint64) (ReleaseFunc, error)
}

var _ ResourceLimiter = ResourceLimiterFunc(nil)

// ReleaseFunc is called when resources should be released after limiting.
//
// RelaseFunc values are never nil values, even in the error case, for
// safety. Users should unconditionally defer these.
type ReleaseFunc func()

// ResourceLimiterFunc is an easy way to construct ResourceLimiters.
type ResourceLimiterFunc func(ctx context.Context, value uint64) (ReleaseFunc, error)

// Acquire implements ResourceLimiter.
func (f ResourceLimiterFunc) Acquire(ctx context.Context, value uint64) (ReleaseFunc, error) {
	if f == nil {
		return func() {}, nil
	}
	return f(ctx, value)
}

// RateLimiter is an interface that components can use to apply
// rate limiting (e.g., network-bytes-per-second, requests-per-second).
type RateLimiter interface {
	// Limit attempts to apply rate limiting based on the provided weight value.
	// Limit is expected to block the caller until the weight can be admitted.
	Limit(ctx context.Context, value uint64) error
}

var _ RateLimiter = RateLimiterFunc(nil)

// RateLimiterFunc is an easy way to construct RateLimiters.
type RateLimiterFunc func(ctx context.Context, value uint64) error

// Limit implements RateLimiter.
func (f RateLimiterFunc) Limit(ctx context.Context, value uint64) error {
	if f == nil {
		return nil
	}
	return f(ctx, value)
}
