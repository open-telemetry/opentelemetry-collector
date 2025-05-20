// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiterhelper // import "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/extension/extensionlimiter"
)

// MultiBaseLimiter returns MustDeny when any element returns MustDeny.
type MultiBaseLimiter []extensionlimiter.BaseLimiter

var _ extensionlimiter.BaseLimiter = MultiBaseLimiter{}

// MustDeny implements BaseLimiter.
func (ls MultiBaseLimiter) MustDeny(ctx context.Context) error {
	var err error
	for _, lim := range ls {
		if lim == nil {
			continue
		}
		err = errors.Join(err, lim.MustDeny(ctx))
	}
	return err
}

// BaseToRateLimiterProvider allows a base limiter to act as a rate
// limiter.  This allows a base limiter to apply to individual Read()
// calls.
func BaseToRateLimiterProvider(blimp extensionlimiter.BaseLimiterProvider) (extensionlimiter.RateLimiterProvider, error) {
	return struct {
		extensionlimiter.GetBaseLimiterFunc
		extensionlimiter.GetRateLimiterFunc
	}{
		blimp.GetBaseLimiter,
		func(_ extensionlimiter.WeightKey, opts ...extensionlimiter.Option) (extensionlimiter.RateLimiter, error) {
			base, err := blimp.GetBaseLimiter(opts...)
			if err != nil {
				return nil, err
			}
			return extensionlimiter.ReserveRateFunc(
				func(ctx context.Context, _ int) (extensionlimiter.RateReservation, error) {
					if err := base.MustDeny(ctx); err != nil {
						return nil, err
					}
					return struct {
						extensionlimiter.WaitTimeFunc
						extensionlimiter.CancelFunc
					}{
						extensionlimiter.WaitTimeFunc(nil),
						extensionlimiter.CancelFunc(nil),
					}, nil
				}), nil
		},
	}, nil
}
