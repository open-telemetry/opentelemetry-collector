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
	BaseLimiterProvider

	GetResourceLimiter(WeightKey, ...Option) (ResourceLimiter, error)
}

// GetResourceLimiterFunc is a functional way to construct
// GetResourceLimiter functions.
type GetResourceLimiterFunc func(WeightKey, ...Option) (ResourceLimiter, error)

// GetResourceLimiter implements part of ResourceLimiterProvider.
func (f GetResourceLimiterFunc) GetResourceLimiter(key WeightKey, opts ...Option) (ResourceLimiter, error) {
	if f == nil {
		return nil, nil
	}
	return f(key, opts...)
}

var _ ResourceLimiterProvider = struct {
	GetResourceLimiterFunc
	GetBaseLimiterFunc
}{}

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
	// ReserveRate is modeled on pkg.go.dev/golang.org/x/time/rate#Limiter.ReserveN,
	// without the time dimension.
	//
	// This is a non-blocking interface; use this interface for
	// callers that cannot be blocked but will instead schedule a
	// resume after DelayFrom(). The context is provided for
	// access to instrumentation and client metadata; the Context
	// deadline is not used.
	ReserveResource(context.Context, uint64) (ResourceReservation, error)

	// WaitForRate is modeled on pkg.go.dev/golang.org/x/time/rate#Limiter.WaitN,
	// without the time dimension.
	//
	// This is a blocking interface. Use this interface for
	// callers that are constrained by a Context deadline, which
	// will be incorporated into the limiter decision.
	WaitForResource(context.Context, uint64) (ReleaseFunc, error)
}

// ResourceReservation is modeled on pkg.go.dev/golang.org/x/time/rate#Reservation
// without the time dimension.
type ResourceReservation interface {
	// Delay returns a channel that will be closed when the
	// reservation request is granted. This resembles a context
	// Done channel.
	Delay() <-chan struct{}

	// Release is called after finishing with the reservation,
	// whether the delay has been reached or not.
	Release()
}

var _ ResourceReservation = struct {
	DelayFunc
	ReleaseFunc
}{}

// ReleaseFunc is called when resources have been released after use.
//
// RelaseFunc values are never nil values, even in the error case for
// safety. Users typically will immediately defer a call to this.
type ReleaseFunc func()

// Release calls this function.
func (f ReleaseFunc) Release() {
	if f == nil {
		return
	}
	f()
}

// DelayFunc returns a channel that is closed when the request is
// permitted to go ahead.
type DelayFunc func() <-chan struct{}

// Delay calls this function.
func (f DelayFunc) Delay() <-chan struct{} {
	if f == nil {
		return immediateChan
	}
	return f()
}

// immediateChan is a singleton channel, already closed, used when a DelayFunc is nil.
var immediateChan = func() <-chan struct{} {
	ic := make(chan struct{})
	close(ic)
	return ic
}()

// ReserveResourceFunc is a functional way to construct ReserveResource interface methods.
type ReserveResourceFunc func(ctx context.Context, value uint64) (ResourceReservation, error)

// ReserveResource implements a ReserveResource interface method.
func (f ReserveResourceFunc) ReserveResource(ctx context.Context, value uint64) (ResourceReservation, error) {
	if f == nil {
		return struct {
			DelayFunc
			ReleaseFunc
		}{
			DelayFunc(nil),
			ReleaseFunc(nil),
		}, nil
	}
	return f(ctx, value)
}

// WaitForResourceFunc is a functional way to construct WaitForResource interface methods.
type WaitForResourceFunc func(context.Context, uint64) (ReleaseFunc, error)

// WaitForResource implements a WaitForResource interface method.
func (f WaitForResourceFunc) WaitForResource(ctx context.Context, value uint64) (ReleaseFunc, error) {
	if f == nil {
		return ReleaseFunc(nil), nil
	}
	return f(ctx, value)
}

var _ ResourceLimiter = struct {
	ReserveResourceFunc
	WaitForResourceFunc
}{}
