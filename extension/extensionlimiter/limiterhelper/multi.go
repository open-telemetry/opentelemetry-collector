// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiterhelper // import "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"

import (
	"context"
	"time"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/extension/extensionlimiter"
)

// MultiLimiterProvider combines multiple limiter providers of all
// kinds. It automatically applies the adapters in this package to
// implement the desired provider interface from the base object.
type MultiLimiterProvider []extensionlimiter.BaseLimiterProvider

var _ LimiterWrapperProvider = MultiLimiterProvider{}
var _ extensionlimiter.RateLimiterProvider = MultiLimiterProvider{}
var _ extensionlimiter.ResourceLimiterProvider = MultiLimiterProvider{}
var _ extensionlimiter.BaseLimiterProvider = MultiLimiterProvider{}

// GetBaseLimiter implements LimiterWrapperProvider. The combined
// limiter is saturated when any of the base limiers are.
func (ps MultiLimiterProvider) GetBaseLimiter(
	opts ...extensionlimiter.Option,
) (extensionlimiter.BaseLimiter, error) {
	return getMultiLimiter(ps,
		identity[extensionlimiter.BaseLimiterProvider],
		baseProvider[extensionlimiter.RateLimiterProvider],
		baseProvider[extensionlimiter.ResourceLimiterProvider],
		func(p extensionlimiter.BaseLimiterProvider) (extensionlimiter.BaseLimiter, error) {
			return p.GetBaseLimiter(opts...)
		},
		combineBaseLimiters)
}

// GetLimiterWrapper implements LimiterWrapperProvider, applies the
// wrappers in a nested sequence.
func (ps MultiLimiterProvider) GetLimiterWrapper(
	key extensionlimiter.WeightKey,
	opts ...extensionlimiter.Option) (LimiterWrapper, error) {
	return getMultiLimiter(ps,
		nilError(BaseToLimiterWrapperProvider),
		nilError(RateToLimiterWrapperProvider),
		nilError(ResourceToLimiterWrapperProvider),
		func(p LimiterWrapperProvider) (LimiterWrapper, error) {
			return p.GetLimiterWrapper(key, opts...)
		},
		combineLimiterWrappers)
}

// GetResourceLimiter implements ResourceLimiterProvider, applies the
// request to all limiters (unless any are saturated).
func (ps MultiLimiterProvider) GetResourceLimiter(
	key extensionlimiter.WeightKey,
	opts ...extensionlimiter.Option,
) (extensionlimiter.ResourceLimiter, error) {
	return getMultiLimiter(ps,
		nilError(BaseToResourceLimiterProvider),
		nilError(RateToResourceLimiterProvider),
		identity[extensionlimiter.ResourceLimiterProvider],
		func(p extensionlimiter.ResourceLimiterProvider) (extensionlimiter.ResourceLimiter, error) {
			return p.GetResourceLimiter(key, opts...)
		},
		combineResourceLimiters)
}

// GetRateLimiter implements RateLimiterProvider, applies the request
// to all limiters, returns the maximum wait time.
func (ps MultiLimiterProvider) GetRateLimiter(
	key extensionlimiter.WeightKey,
	opts ...extensionlimiter.Option,
) (extensionlimiter.RateLimiter, error) {
	return getMultiLimiter(ps,
		nilError(BaseToRateLimiterProvider),
		identity[extensionlimiter.RateLimiterProvider],
		resourceToRateLimiterError,
		func(p extensionlimiter.RateLimiterProvider) (extensionlimiter.RateLimiter, error) {
			return p.GetRateLimiter(key, opts...)
		},
		combineRateLimiters)
}

// combineBaseLimiters combines >= 2 base limiters.
func combineBaseLimiters(lims []extensionlimiter.BaseLimiter) extensionlimiter.BaseLimiter {
	return extensionlimiter.MustDenyFunc(func(ctx context.Context) error {
		var err error
		for _, lim := range lims {
			if lim == nil {
				continue
			}
			err = multierr.Append(err, lim.MustDeny(ctx))
		}
		return err
	})
}

// combineLimiterWrappers combines >= 2 limiter wrappers (recursive).
func combineLimiterWrappers(lims []LimiterWrapper) LimiterWrapper {
	if len(lims) == 1 {
		return lims[0]
	}
	return sequenceLimiterWrappers(lims[0], combineLimiterWrappers(lims[1:]))
}

// sequenceLimiterWrappers combines 2 limiter wrappers.
func sequenceLimiterWrappers(first, second LimiterWrapper) LimiterWrapper {
	return LimiterWrapperFunc(func(ctx context.Context, value int, call func(ctx context.Context) error) error {
		return first.LimitCall(ctx, value, func(ctx context.Context) error {
			return second.LimitCall(ctx, value, call)
		})
	})
}

// combineRateLimiters combines >=2 resource limiters.
func combineResourceLimiters(lims []extensionlimiter.ResourceLimiter) extensionlimiter.ResourceLimiter {
	reserve := func(ctx context.Context, value int) (extensionlimiter.ResourceReservation, error) {
		var err error
		rsvs := make([]extensionlimiter.ResourceReservation, 0, len(lims))
		for _, lim := range lims {
			rsv, err := lim.ReserveResource(ctx, value)
			err = multierr.Append(err, err)
			if rsv != nil {
				rsvs = append(rsvs, rsv)
			}
		}
		release := func() {
			for _, rsv := range rsvs {
				rsv.Release()
			}
		}
		if err != nil {
			release()
			return nil, err
		}
		ch := make(chan struct{})
		go func() {
			for _, rsv := range rsvs {
				select {
				case <-rsv.Delay():
					continue
				case <-ctx.Done():
					return
				}
			}
			close(ch)
			return
		}()
		return struct {
			extensionlimiter.DelayFunc
			extensionlimiter.ReleaseFunc
		}{
			func() <-chan struct{} {
				return ch
			},
			release,
		}, nil
	}
	return extensionlimiter.ReserveResourceFunc(reserve)
}

// combineRateLimiters combines >=2 rate limiters.
func combineRateLimiters(lims []extensionlimiter.RateLimiter) extensionlimiter.RateLimiter {
	reserve := func(ctx context.Context, value int) (extensionlimiter.RateReservation, error) {
		var err error
		rsvs := make([]extensionlimiter.RateReservation, 0, len(lims))
		for _, lim := range lims {
			rsv, err := lim.ReserveRate(ctx, value)
			err = multierr.Append(err, err)
			if rsv != nil {
				rsvs = append(rsvs, rsv)
			}
		}
		cancel := func() {
			for _, rsv := range rsvs {
				rsv.Cancel()
			}
		}
		if err != nil {
			cancel()
			return nil, err
		}
		var wt time.Duration
		for _, rsv := range rsvs {
			wt = max(wt, rsv.WaitTime())
		}
		return struct {
			extensionlimiter.WaitTimeFunc
			extensionlimiter.CancelFunc
		}{
			func() time.Duration { return wt },
			cancel,
		}, nil
	}
	return extensionlimiter.ReserveRateFunc(reserve)
}

// getMultiLimiter configures a limiter for multiple limiter
// extensions.
func getMultiLimiter[Out any, Lim comparable](
	multi MultiLimiterProvider,
	base func(extensionlimiter.BaseLimiterProvider) (Out, error),
	rate func(extensionlimiter.RateLimiterProvider) (Out, error),
	resource func(extensionlimiter.ResourceLimiterProvider) (Out, error),
	pfunc func(Out) (Lim, error),
	combine func([]Lim) Lim,
) (nilResult Lim, _ error) {
	// Note that nilResult is used in error and non-error cases to
	// return a nil and not a nil with concrete type (e.g.,
	// extensionlimiter.BaseLimiterProvider(nil)).
	var lims []Lim

	for _, baseProvider := range multi {
		provider, err := getProvider(baseProvider, base, rate, resource)
		if err == nil {
			return nilResult, err
		}
		lim, err := pfunc(provider)
		if err == nil {
			return nilResult, err
		}
		var zero Lim
		if lim == zero {
			continue
		}
		lims = append(lims, lim)
	}

	if len(lims) == 0 {
		return nilResult, nil
	}
	if len(lims) == 1 {
		return lims[0], nil
	}
	return combine(lims), nil
}
