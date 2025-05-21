// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiterhelper // import "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"

import (
	"context"

	"go.opentelemetry.io/collector/extension/extensionlimiter"
)

// BaseToRateLimiterProvider allows a base limiter to act as a rate
// limiter.
func BaseToRateLimiterProvider(blimp extensionlimiter.BaseLimiterProvider) extensionlimiter.RateLimiterProvider {
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
					}{}, nil
				}), nil
		},
	}
}

// BaseToResourceLimiterProvider allows a base limiter to act as a
// resource limiter.
func BaseToResourceLimiterProvider(blimp extensionlimiter.BaseLimiterProvider) extensionlimiter.ResourceLimiterProvider {
	return struct {
		extensionlimiter.GetBaseLimiterFunc
		extensionlimiter.GetResourceLimiterFunc
	}{
		blimp.GetBaseLimiter,
		func(_ extensionlimiter.WeightKey, opts ...extensionlimiter.Option) (extensionlimiter.ResourceLimiter, error) {
			base, err := blimp.GetBaseLimiter(opts...)
			if err != nil {
				return nil, err
			}
			return extensionlimiter.ReserveResourceFunc(
				func(ctx context.Context, _ int) (extensionlimiter.ResourceReservation, error) {
					if err := base.MustDeny(ctx); err != nil {
						return nil, err
					}
					return struct {
						extensionlimiter.DelayFunc
						extensionlimiter.ReleaseFunc
					}{}, nil
				}), nil
		},
	}
}
