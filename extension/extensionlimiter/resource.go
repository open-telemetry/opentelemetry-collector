// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

import (
	"context"
)

// ResourceLimiterProvider is a provider for resource limiters.
//
// Limiter implementations will implement this or the
// RateLimiterProvider interface, but MUST not implement both.
// Limiters are covered by configmiddleware configuration, which
// is able to construct LimiterWrappers from these providers.
type ResourceLimiterProvider interface {
	ResourceLimiter(WeightKey) (ResourceLimiter, error)
}

// ResourceLimiterProviderFunc is a functional way to build ResourceLimters.
type ResourceLimiterProviderFunc func(WeightKey) (ResourceLimiter, error)

var _ ResourceLimiterProvider = ResourceLimiterProviderFunc(nil)

// ResourceLimiter implements ResourceLimiterProvider.
func (f ResourceLimiterProviderFunc) ResourceLimiter(key WeightKey) (ResourceLimiter, error) {
	return f(key)
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
	// Limiter includes MustDeny().
	Limiter

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
	// when the resources are no longer needed.
	//
	// Implementations are not required to call a release func
	// when Acquire(0) is called, because there is nothing to
	// release. Acquire(0) the equivalent of MustDeny().
	Acquire(ctx context.Context, value uint64) (ReleaseFunc, error)
}

// ReleaseFunc is called when resources should be released after limiting.
//
// RelaseFunc values are never nil values, even in the error case, for
// safety. Users should unconditionally defer these.
type ReleaseFunc func()

// ResourceLimiterFunc is a functional way to construct ResourceLimiters.
type ResourceLimiterFunc func(ctx context.Context, value uint64) (ReleaseFunc, error)

var _ ResourceLimiter = ResourceLimiterFunc(nil)

// MustDeny implements ResourceLimiter
func (f ResourceLimiterFunc) MustDeny(ctx context.Context) error {
	_, err := f.Acquire(ctx, 0)
	return err
}

// Acquire implements ResourceLimiter
func (f ResourceLimiterFunc) Acquire(ctx context.Context, value uint64) (ReleaseFunc, error) {
	if f == nil {
		return func() {}, nil
	}
	return f(ctx, value)
}

// NewResourceLimiterWrapper returns a LimiterWrapper from a ResourceLimiter.
func NewResourceLimiterWrapper(limiter ResourceLimiter) LimiterWrapper {
	return resourceLimiterWrapper{limiter: limiter}
}

type resourceLimiterWrapper struct {
	limiter ResourceLimiter
}

var _ LimiterWrapper = resourceLimiterWrapper{}

// MustDeny implements LimiterWrapper.
func (w resourceLimiterWrapper) MustDeny(ctx context.Context) error {
	if w.limiter == nil {
		return nil
	}
	return w.limiter.MustDeny(ctx)
}

// LimitCall implements LimiterWrapper.
func (w resourceLimiterWrapper) LimitCall(ctx context.Context, value uint64, call func(context.Context) error) error {
	release, err := w.limiter.Acquire(ctx, value)
	if err != nil {
		return err
	}
	defer release()
	return call(ctx)
}
