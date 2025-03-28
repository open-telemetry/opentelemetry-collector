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

// Provider is an interface that provides access to different limiter types
// for specific weight keys.
type Provider interface {
	// RateLimiter returns a RateLimiter for the specified weight key
	RateLimiter(key WeightKey) RateLimiter

	// ResourceLimiter returns a ResourceLimiter for the specified weight key.
	//
	// In cases where a component supports a rate limiter and does not use
	// a release function, the component may return a ResourceLimiterFunc
	// which calls the underlying rate limiter and returns a nil ReleaseFunc.
	ResourceLimiter(key WeightKey) ResourceLimiter
}

// releaseFuncs is a collection of release functions that can be called together
type releaseFuncs []ReleaseFunc

// release calls all non-nil release functions in the collection
func (r releaseFuncs) release() {
	for _, rf := range r {
		if rf != nil {
			rf()
		}
	}
}

// MultiProvider combines multiple Provider implementations into a single Provider.
// When requesting a limiter, it returns a combined limiter that applies all
// limits from the underlying providers.
type MultiProvider struct {
	providers []Provider
}

// NewMultiProvider creates a new MultiProvider from the given providers.
func NewMultiProvider(providers ...Provider) *MultiProvider {
	return &MultiProvider{providers: providers}
}

// RateLimiter returns a combined RateLimiter for the specified weight key.
// The combined limiter will apply all limits from the underlying providers.
func (mp *MultiProvider) RateLimiter(key WeightKey) RateLimiter {
	limiters := make([]RateLimiter, 0, len(mp.providers))
	for _, p := range mp.providers {
		if limiter := p.RateLimiter(key); limiter != nil {
			limiters = append(limiters, limiter)
		}
	}

	if len(limiters) == 0 {
		return nil
	}

	return RateLimiterFunc(func(ctx context.Context, value uint64) error {
		for _, limiter := range limiters {
			if err := limiter.Limit(ctx, value); err != nil {
				return err
			}
		}
		return nil
	})
}

// ResourceLimiter returns a combined ResourceLimiter for the specified weight key.
// The combined limiter will apply all limits from the underlying providers and
// aggregate the release functions.
func (mp *MultiProvider) ResourceLimiter(key WeightKey) ResourceLimiter {
	limiters := make([]ResourceLimiter, 0, len(mp.providers))
	for _, p := range mp.providers {
		if limiter := p.ResourceLimiter(key); limiter != nil {
			limiters = append(limiters, limiter)
		}
	}

	if len(limiters) == 0 {
		return nil
	}

	return ResourceLimiterFunc(func(ctx context.Context, value uint64) (ReleaseFunc, error) {
		var funcs releaseFuncs

		for _, limiter := range limiters {
			releaseFunc, err := limiter.Acquire(ctx, value)
			if err != nil {
				// Release any already acquired resources
				funcs.release()
				return nil, err
			}
			if releaseFunc != nil {
				funcs = append(funcs, releaseFunc)
			}
		}

		if len(funcs) == 0 {
			return nil, nil
		}

		return funcs.release, nil
	})
}

// ReleaseFunc is called when resources should be released after limiting.
//
// Note that RelaseFunc values may be nil in cases where the implementation is
// not concerned with releasing acquired resources, such as when a rate limiter
// receives the Acquire() signal for requests.
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
