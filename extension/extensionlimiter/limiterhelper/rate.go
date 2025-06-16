// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiterhelper // import "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"

import (
	"context"
	"errors"
	"time"

	"golang.org/x/time/rate"

	"go.opentelemetry.io/collector/extension/extensionlimiter"
)

var (
	ErrRateLimitExceeded   = errors.New("rate limit is saturated")
	ErrRateRequestTooLarge = errors.New("rate request exceeds burst size")
	ErrRateRequestDeadline = errors.New("rate request exceeds context deadline")
)

// NewRateLimiter returns an implementation of the rate-limiter
// extension based on logic from x/time/rate.
func NewRateLimiter(frequency float64, burst int) extensionlimiter.RateLimiter {
	limit := rate.Limit(frequency)
	limiter := rate.NewLimiter(limit, burst)

	reserve := func(ctx context.Context, value int) (extensionlimiter.RateReservation, error) {
		// Check if context was canceled.
		select {
		case <-ctx.Done():
			return nil, context.Cause(ctx)
		default:
		}

		// Check for out-of-range requests.
		if value > burst && limit != rate.Inf {
			return nil, ErrRateRequestTooLarge
		}

		// Call the non-blocking API.
		rsv := limiter.ReserveN(time.Now(), value)
		if !rsv.OK() {
			return nil, ErrRateLimitExceeded
		}

		// Compare the wait and deadline.
		when := time.Now()
		wait := rsv.DelayFrom(when)
		if deadline, ok := ctx.Deadline(); ok {
			if deadline.Sub(when) < wait {
				rsv.Cancel()
				return nil, ErrRateRequestDeadline
			}
		}

		// Return the wait time and cancel function.
		return struct {
			extensionlimiter.WaitTimeFunc
			extensionlimiter.CancelFunc
		}{
			func() time.Duration { return wait },
			rsv.Cancel,
		}, nil
	}

	return extensionlimiter.ReserveRateFunc(reserve)
}

// BlockingRateLimiter wraps a RateLimiter extension in a blocking
// interface which takes the context deadline into account before
// waiting for the rate allowance.
type BlockingRateLimiter struct {
	limiter extensionlimiter.RateLimiter
}

// NewBlockingRateLimiter returns a blocking wrapper for RateLimiter
// extensions.
func NewBlockingRateLimiter(limiter extensionlimiter.RateLimiter) BlockingRateLimiter {
	return BlockingRateLimiter{
		limiter: limiter,
	}
}

// WaitFor blocks the caller until the requested value is allowed by
// the limiter.
func (b BlockingRateLimiter) WaitFor(ctx context.Context, value int) error {
	newTimer := func(d time.Duration) (<-chan time.Time, func() bool, func()) {
		timer := time.NewTimer(d)
		return timer.C, timer.Stop, func() {}
	}
	return b.waitFor(ctx, value, newTimer)
}

// timerFunc is a test helper for testing the blocking rate limter.
type timerFunc func(time.Duration) (<-chan time.Time, func() bool, func())

// waitFor is a testable form of BlockingRateLimiter.WaitFor.
func (b BlockingRateLimiter) waitFor(ctx context.Context, value int, timer timerFunc) error {
	// Reserve from the underlying limiter.
	rsv, err := b.limiter.ReserveRate(ctx, value)
	if err != nil {
		return err
	}

	// Wait, if necessary.
	delay := rsv.WaitTime()
	if delay == 0 {
		return nil
	}

	ch, stop, advance := timer(delay)
	defer stop()
	advance() // only has an effect when testing
	select {
	case <-ch:
		// Proceed. Do not cancel.
		return nil

	case <-ctx.Done():
		// Context was canceled before we could proceed.
		rsv.Cancel()
		return context.Cause(ctx)
	}
}

// RateToResourceLimiterProvider allows a rate limiter to act as a
// resource limiter. Note that the opposite direction (i.e., resource
// limter acting as rate limiter) is an invalid configuration.
func RateToResourceLimiterProvider(blimp extensionlimiter.RateLimiterProvider) extensionlimiter.ResourceLimiterProvider {
	return struct {
		extensionlimiter.GetSaturationCheckerFunc
		extensionlimiter.GetResourceLimiterFunc
	}{
		blimp.GetSaturationChecker,
		func(weight extensionlimiter.WeightKey, opts ...extensionlimiter.Option) (extensionlimiter.ResourceLimiter, error) {
			rlim, err := blimp.GetRateLimiter(weight, opts...)
			if err != nil {
				return nil, err
			}
			return extensionlimiter.ReserveResourceFunc(
				func(ctx context.Context, value int) (extensionlimiter.ResourceReservation, error) {
					rsv, err := rlim.ReserveRate(ctx, value)
					if err != nil {
						return nil, err
					}
					cch := make(chan struct{})
					timer := time.AfterFunc(rsv.WaitTime(), func() {
						close(cch)
					})
					return struct {
						extensionlimiter.DelayFunc
						extensionlimiter.ReleaseFunc
					}{
						func() <-chan struct{} { return cch },
						func() {
							select {
							case <-cch:
								// The timer fired
							default:
								rsv.Cancel()
								timer.Stop()
							}
						},
					}, nil
				}), nil
		},
	}
}
