// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

import (
	"context"
)

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
	// resume after Delay(). The context is provided for
	// access to instrumentation and client metadata; the Context
	// deadline is not used.
	ReserveResource(context.Context, int) (ResourceReservation, error)
}

// ReserveResourceFunc is a functional way to construct ReserveResource interface methods.
type ReserveResourceFunc func(ctx context.Context, value int) (ResourceReservation, error)

// ReserveResource implements a ReserveResource interface method.
func (f ReserveResourceFunc) ReserveResource(ctx context.Context, value int) (ResourceReservation, error) {
	if f == nil {
		return NewResourceReservationImpl(nil, nil), nil
	}
	return f(ctx, value)
}

// resourceLimiterImpl is a functional ResourceLimiter object.  The zero state
// is a no-op.
type resourceLimiterImpl struct {
	ReserveResourceFunc
}

var _ ResourceLimiter = resourceLimiterImpl{}

// NewResourceLimiterImpl returns a functional implementation of
// ResourceLimiter.  Use a nil argument for the no-op implementation.
func NewResourceLimiterImpl(f ReserveResourceFunc) ResourceLimiter {
	return resourceLimiterImpl{
		ReserveResourceFunc: f,
	}
}
