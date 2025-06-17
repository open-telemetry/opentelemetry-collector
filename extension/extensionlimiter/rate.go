// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

import (
	"context"
	"time"
)

// RateLimiterProvider is a provider for rate limiters.
//
// Limiter implementations will implement this or the
// ResourceLimiterProvider interface, but MUST not implement both.
// Limiters are covered by configmiddleware configuration, which is
// able to construct LimiterWrappers from these providers.
type RateLimiterProvider interface {
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

var _ RateLimiterProvider = GetRateLimiterFunc(nil)

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

// RateReservation is modeled on pkg.go.dev/golang.org/x/time/rate#Reservation
type RateReservation interface {
	// WaitTime returns the duration until this reservation may
	// proceed. A typical implementation uses Reservation.DelayFrom(time.Now()),
	// and callers typically/ use time.After or time.Timer to implement a delay.
	WaitTime() time.Duration

	// Cancel cancels the reservation before it is used. A typical
	// implementation uses Reservation.CancelAt(time.Now()).
	Cancel()
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

// WaitTimeFunc is a functional way to construct WaitTime functions.
type WaitTimeFunc func() time.Duration

// WaitTime implements part of Reservation.
func (f WaitTimeFunc) WaitTime() time.Duration {
	if f == nil {
		return 0
	}
	return f.WaitTime()
}

// CancelFunc is a functional way to construct Cancel functions.
type CancelFunc func()

// Reserve implements part of the RateReserveer interface.
func (f CancelFunc) Cancel() {
	if f == nil {
		return
	}
	f.Cancel()
}

func NewRateLimiterProviderImpl(f GetRateLimiterFunc) RateLimiterProvider {
	return f
}

func NewRateLimiterImpl(f ReserveRateFunc) RateLimiter {
	return f 
}

// rateReservationImpl is a struct that implements RateReservation.
// The zero state is a no-op.
type rateReservationImpl struct {
	WaitTimeFunc
	CancelFunc
}

var _ RateReservation = rateReservationImpl{}

func NewRateReservationImpl(wf WaitTimeFunc, cf CancelFunc) RateReservation {
	return rateReservationImpl{
		WaitTimeFunc: wf,
		CancelFunc: cf,
	}
}

func NewNopRateReservation() RateReservation {
	return rateReservationImpl{}
}
