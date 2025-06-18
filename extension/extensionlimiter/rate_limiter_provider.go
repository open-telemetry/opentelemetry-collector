// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

// RateLimiterProvider is a provider for rate limiters.
//
// Limiter implementations will implement this or the
// ResourceLimiterProvider interface, but MUST not implement both.
// Limiters are covered by configmiddleware configuration, which is
// able to construct LimiterWrappers from these providers.
type RateLimiterProvider interface {
	// AnyProvider is provided by embedding AnyProviderImpl.
	AnyProvider
	
	// GetRateLimiter returns a rate limiter for a weight key.
	GetRateLimiter(WeightKey, ...Option) (RateLimiter, error)
}

// GetRateLimiterFunc is a functional way to construct GetRateLimiter
// functions.
type GetRateLimiterFunc func(WeightKey, ...Option) (RateLimiter, error)

// RateLimiter implements RateLimiterProvider.
func (f GetRateLimiterFunc) GetRateLimiter(key WeightKey, opts ...Option) (RateLimiter, error) {
	if f == nil {
		return NewRateLimiterImpl(nil), nil
	}
	return f(key, opts...)
}

type rateLimiterProviderImpl struct {
	AnyProviderImpl

	GetRateLimiterFunc
}

var _ RateLimiterProvider = rateLimiterProviderImpl{}

// NewRateLimiterProviderImpl returns a functional implementation of
// RateLimiterProvider.  Use a nil argument for the no-op implementation.
func NewRateLimiterProviderImpl(f GetRateLimiterFunc) RateLimiterProvider {
	return rateLimiterProviderImpl{
		GetRateLimiterFunc: f,
	}
}
