// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

import (
	"context"
)

// Defines:
// - ResourceLimiterProvider
// - ResourceLimiterProviderFunc
// - ResourceLimiter
// - ResourceLimiterFunc
// - ReleaseFunc

// ResourceLimiterProvider is a provider for resource limiters.
//
// Limiter implementations will implement this or the
// RateLimiterProvider interface, but MUST not implement both.
// Limiters are covered by configmiddleware configuration, which
// is able to construct LimiterWrappers from these providers.
type ResourceLimiterProvider interface {
	ResourceLimiter(WeightKey, ...Option) (ResourceLimiter, error)
}

// ResourceLimiterProviderFunc is a functional way to build ResourceLimters.
type ResourceLimiterProviderFunc func(WeightKey, ...Option) (ResourceLimiter, error)

var _ ResourceLimiterProvider = ResourceLimiterProviderFunc(nil)

// ResourceLimiter implements ResourceLimiterProvider.
func (f ResourceLimiterProviderFunc) ResourceLimiter(key WeightKey, opts ...Option) (ResourceLimiter, error) {
	return f(key, opts...)
}

// ResourceLimiter is an interface that an implementation makes
// available to apply physical limits on quantities such as the number
// of concurrent requests or amount of memory in use.
//
// This is a relatively low-level interface. Callers that can use a
// LimiterWrapper should choose that interface instead.  This
// interface is meant for direct use only in special cases where
// control flow is not scoped to a callback, for example in a
// streaming receiver where a limiter might be Acquired in the body of
// Send() and released prior to a corresponding Recv() (e.g.,
// OTel-Arrow receiver).
//
// See the README for more recommendations.
type ResourceLimiter interface {
	// Acquire attempts to acquire a quantified resource with the
	// provided weight, based on the key that was given to the
	// provider. The caller has these options:
	//
	// - Accept and let the request proceed by returning a release func and a nil error
	// - Fail and return a non-nil error and a nil release func
	// - Block until the resource becomes available, then accept
	// - Block until the context times out, return the error.
	//
	// See the README for more recommendations.
	//
	// On success, it returns a ReleaseFunc that should be called
	// after the resources is no longer in use.
	Acquire(ctx context.Context, value uint64) (ReleaseFunc, error)
}

// ReleaseFunc is called when resources have been released after use.
//
// RelaseFunc values are never nil values, even in the error case, for
// safety. Users may unconditionally defer these.
//
// Implementations are not required to call a release func after
// Acquire(0) is called, since there is nothing to release.
type ReleaseFunc func()

// ResourceLimiterFunc is a functional way to construct ResourceLimiters.
type ResourceLimiterFunc func(ctx context.Context, value uint64) (ReleaseFunc, error)

var _ ResourceLimiter = ResourceLimiterFunc(nil)

// Acquire implements ResourceLimiter
func (f ResourceLimiterFunc) Acquire(ctx context.Context, value uint64) (ReleaseFunc, error) {
	if f == nil {
		return func() {}, nil
	}
	return f(ctx, value)
}
