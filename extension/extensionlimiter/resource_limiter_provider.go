// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

// ResourceLimiterProvider is a provider for resource limiters.
//
// Limiter implementations will implement this or the
// RateLimiterProvider interface, but MUST not implement both.
// Limiters are covered by configmiddleware configuration, which
// is able to construct LimiterWrappers from these providers.
type ResourceLimiterProvider interface {
	GetResourceLimiter(WeightKey, ...Option) (ResourceLimiter, error)
}

// GetResourceLimiterFunc is a functional way to construct
// GetResourceLimiter functions.
type GetResourceLimiterFunc func(WeightKey, ...Option) (ResourceLimiter, error)

// GetResourceLimiter implements part of ResourceLimiterProvider.
func (f GetResourceLimiterFunc) GetResourceLimiter(key WeightKey, opts ...Option) (ResourceLimiter, error) {
	if f == nil {
		return NewResourceLimiterImpl(nil), nil
	}
	return f(key, opts...)
}


type resourceLimiterProviderImpl struct {
	GetResourceLimiterFunc
}

var _ ResourceLimiterProvider = resourceLimiterProviderImpl{}

// NewResourceLimiterProviderImpl returns a functional implementation of
// ResourceLimiterProvider.  Use a nil argument for the no-op implementation.
func NewResourceLimiterProviderImpl(f GetResourceLimiterFunc) ResourceLimiterProvider {
	return resourceLimiterProviderImpl{
		GetResourceLimiterFunc: f,
	}
}
