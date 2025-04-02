// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiterhelper // import "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/extension/extensionlimiter"
)

// MiddlewaresToLimiterProvider returns a multi-provider that combines all middleware limiter providers.
func MiddlewaresToLimiterProvider(host component.Host, middlewares []configmiddleware.Middleware) extensionlimiter.Provider {
	var providers []extensionlimiter.Provider
	exts := host.GetExtensions()
	for _, middleware := range middlewares {
		ext := exts[middleware.MiddlewareID]
		if ext == nil {
			continue
		}
		if provider, ok := ext.(extensionlimiter.Provider); ok {
			providers = append(providers, provider)
		}
	}
	return NewMultiProvider(providers...)
}

// MultiProvider combines multiple Provider implementations into a single Provider.
// When requesting a limiter, it returns a combined limiter that applies all
// limits from the underlying providers.
type MultiProvider struct {
	providers []extensionlimiter.Provider
}

// NewMultiProvider creates a new MultiProvider from the given providers.
func NewMultiProvider(providers ...extensionlimiter.Provider) *MultiProvider {
	return &MultiProvider{providers: providers}
}

// RateLimiter returns a combined RateLimiter for the specified weight key.
// The combined limiter will apply all limits from the underlying providers.
func (mp *MultiProvider) RateLimiter(key extensionlimiter.WeightKey) extensionlimiter.RateLimiter {
	limiters := make([]extensionlimiter.RateLimiter, 0, len(mp.providers))
	for _, p := range mp.providers {
		if limiter := p.RateLimiter(key); limiter != nil {
			limiters = append(limiters, limiter)
		}
	}

	if len(limiters) == 0 {
		return nil
	}

	return extensionlimiter.RateLimiterFunc(func(ctx context.Context, value uint64) error {
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
func (mp *MultiProvider) ResourceLimiter(key extensionlimiter.WeightKey) extensionlimiter.ResourceLimiter {
	limiters := make([]extensionlimiter.ResourceLimiter, 0, len(mp.providers))
	for _, p := range mp.providers {
		if limiter := p.ResourceLimiter(key); limiter != nil {
			limiters = append(limiters, limiter)
		}
	}

	if len(limiters) == 0 {
		return nil
	}

	return extensionlimiter.ResourceLimiterFunc(func(ctx context.Context, value uint64) (extensionlimiter.ReleaseFunc, error) {
		var funcs releaseFuncs

		for _, limiter := range limiters {
			releaseFunc, err := limiter.Acquire(ctx, value)
			if err != nil {
				// Release any already acquired resources
				funcs.release()
				return func() {}, err
			}
			if releaseFunc != nil {
				funcs = append(funcs, releaseFunc)
			}
		}

		if len(funcs) == 0 {
			return func() {}, nil
		}

		return funcs.release, nil
	})
}

// releaseFuncs is a collection of release functions that can be called together
type releaseFuncs []extensionlimiter.ReleaseFunc

// release calls all non-nil release functions in the collection
func (r releaseFuncs) release() {
	for _, rf := range r {
		if rf != nil {
			rf()
		}
	}
}
