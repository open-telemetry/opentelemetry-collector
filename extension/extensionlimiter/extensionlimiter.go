// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

import (
	"context"
)

// WeightKey is an enum type for common rate limits
type WeightKey string

// Predefined weight keys for common rate limits
const (
	WeightKeyNetworkBytes   WeightKey = "network_bytes"
	WeightKeyRequestItems   WeightKey = "request_items"
	WeightKeyRequestCount   WeightKey = "request_count"
	WeightKeyResidentMemory WeightKey = "resident_memory"
)

// ReleaseFunc is called when resources should be released after limiting.
type ReleaseFunc func()

// ResourceLimiter is an interface that components can use to apply rate limiting.
// Extensions implementing this interface can be referenced by their
// names from component rate limiting configurations.
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

type ResourceLimiterFunc func(ctx context.Context, value uint64) (ReleaseFunc, error)

func (f ResourceLimiterFunc) Acquire(ctx context.Context, value uint64) (ReleaseFunc, error) {
	if f == nil {
		return nil, nil
	}
	return f(ctx, value)
}

// RateLimiter is an interface for rate limiting without resource release.
type RateLimiter interface {
	// Limit attempts to apply rate limiting based on the provided weight value.
	// Limit is expected to block the caller until the weight can be admitted.
	Limit(ctx context.Context, value uint64) error
}

var _ RateLimiter = RateLimiterFunc(nil)

// RateLimiterFunc is a function type that implements RateLimiter interface
type RateLimiterFunc func(ctx context.Context, value uint64) error

func (f RateLimiterFunc) Limit(ctx context.Context, value uint64) error {
	if f == nil {
		return nil
	}
	return f(ctx, value)
}

// Provider is an interface that provides access to different limiter types
// for specific weight keys.
type Provider interface {
	// RateLimiter returns a RateLimiter for the specified weight key
	RateLimiter(key WeightKey) RateLimiter

	// ResourceLimiter returns a ResourceLimiter for the specified weight key
	ResourceLimiter(key WeightKey) ResourceLimiter
}
