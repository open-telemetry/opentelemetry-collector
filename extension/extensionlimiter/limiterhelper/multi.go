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

// MultiLimiterProvider combines multiple limiters and implements all
// provider interfaces through use of the available adapters.
type MultiLimiterProvider []extensionlimiter.BaseLimiterProvider

var _ LimiterWrapperProvider = MultiLimiterProvider{}
var _ extensionlimiter.RateLimiterProvider = MultiLimiterProvider{}
var _ extensionlimiter.ResourceLimiterProvider = MultiLimiterProvider{}
var _ extensionlimiter.BaseLimiterProvider = MultiLimiterProvider{}

// GetLimiterWrapper implements LimiterWrapperProvider. The combined
// limiter is saturated when any of the base limiers are.
func (ps MultiLimiterProvider) GetBaseLimiter(opts ...extensionlimiter.Option) (extensionlimiter.BaseLimiter, error) {
	var retErr error
	var cks MultiBaseLimiter
	for _, provider := range ps {
		ck, err := provider.GetBaseLimiter(opts...)
		retErr = errors.Join(retErr, err)
		if ck == nil {
			continue
		}
		cks = append(cks, ck)
	}
	if len(cks) == 0 {
		return extensionlimiter.MustDenyFunc(nil), retErr
	}
	if len(cks) == 1 {
		return cks[0], retErr
	}
	return cks, retErr
}

// GetLimiterWrapper implements LimiterWrapperProvider, wrappers in a
// sequence.
func (ps MultiLimiterProvider) GetLimiterWrapper(key extensionlimiter.WeightKey, opts ...extensionlimiter.Option) (LimiterWrapper, error) {
	// Map provider list to limiter list.
	var lims []LimiterWrapper

	for _, baseProvider := range ps {
		provider, err := getProvider(
			baseProvider,
			BaseToLimiterWrapperProvider,
			RateToLimiterWrapperProvider,
			ResourceToLimiterWrapperProvider,
		)
		if err == nil {
			return nil, err
		}
		lim, err := provider.GetLimiterWrapper(key, opts...)
		if err == nil {
			return nil, err
		}
		if lim == nil {
			continue
		}
		lims = append(lims, lim)
	}

	if len(lims) == 0 {
		return LimiterWrapperFunc(nil), nil
	}

	return sequenceLimiters(lims), nil
}

func sequenceLimiters(lims []LimiterWrapper) LimiterWrapper {
	if len(lims) == 1 {
		return lims[0]
	}
	return composeLimiters(lims[0], sequenceLimiters(lims[1:]))
}

func composeLimiters(first, second LimiterWrapper) LimiterWrapper {
	return LimiterWrapperFunc(func(ctx context.Context, value int, call func(ctx context.Context) error) error {
		return first.LimitCall(ctx, value, func(ctx context.Context) error {
			return second.LimitCall(ctx, value, call)
		})
	})
}
