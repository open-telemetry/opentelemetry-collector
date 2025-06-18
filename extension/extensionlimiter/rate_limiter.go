// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

import (
	"context"
)

// RateLimiter is an interface that an implementation makes available
// to apply time-based limits on quantities such as the number of
// bytes or items per second.
//
// This is a relatively low-level interface. Callers that can use a
// LimiterWrapper should choose that interface instead. This interface
// is meant for direct use only in special cases where control flow
// cannot be easily scoped to a callback, for example inside
// middleware (e.g., grpc.StatsHandler).
//
// See the README for more recommendations.
type RateLimiter interface {
	// ReserveRate is modeled on pkg.go.dev/golang.org/x/time/rate#Limiter.ReserveN
	//
	// This is a non-blocking interface; use this interface for
	// callers that cannot be blocked but will instead schedule a
	// resume after Delay(). The context is provided for access to
	// instrumentation and client metadata; the Context deadline 
	// is not used, should be considered by the caller.
	ReserveRate(context.Context, int) (RateReservation, error)
}

// ReserveRateFunc is a functional way to construct ReserveRate functions.
type ReserveRateFunc func(context.Context, int) (RateReservation, error)

// Reserve implements part of the RateLimiter interface.
func (f ReserveRateFunc) ReserveRate(ctx context.Context, value int) (RateReservation, error) {
	if f == nil {
		return NewRateReservationImpl(nil, nil), nil
	}
	return f(ctx, value)
}

// rateLimiterImpl is a functional RateLimiter object.  The zero state
// is a no-op.
type rateLimiterImpl struct {
	ReserveRateFunc
}

var _ RateLimiter = rateLimiterImpl{}

// NewRateLimiterImpl returns a functional implementation of
// RateLimiter.  Use a nil argument for the no-op implementation.
func NewRateLimiterImpl(f ReserveRateFunc) RateLimiter {
	return rateLimiterImpl{
		ReserveRateFunc: f,
	}
}
