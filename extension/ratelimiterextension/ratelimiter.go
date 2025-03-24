// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ratelimiterextension

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionlimiter"
	"go.uber.org/zap"
)

// tokenBucket implements a token bucket rate limiter
type tokenBucket struct {
	// rate at which tokens are added to the bucket per second
	rate float64
	// maximum size of the bucket
	burstSize int64
	// current number of tokens in the bucket
	tokens int64
	// last time tokens were added to the bucket
	lastUpdate time.Time
	// mutex to protect the token bucket
	mu sync.Mutex
}

// newTokenBucket creates a new token bucket with the specified rate and burst size
func newTokenBucket(rate float64, burstSize int64) *tokenBucket {
	return &tokenBucket{
		rate:       rate,
		burstSize:  burstSize,
		tokens:     burstSize, // Start with a full bucket
		lastUpdate: time.Now(),
	}
}

// updateTokens adds tokens to the bucket based on elapsed time
func (tb *tokenBucket) updateTokens() {
	now := time.Now()
	elapsed := now.Sub(tb.lastUpdate).Seconds()

	// Calculate tokens to add based on elapsed time and rate
	newTokens := int64(elapsed * tb.rate)

	if newTokens > 0 {
		tb.tokens += newTokens
		if tb.tokens > tb.burstSize {
			tb.tokens = tb.burstSize // Cap at burst size
		}
		tb.lastUpdate = now
	}
}

// tryConsume attempts to consume tokens and returns whether it succeeded
// If it didn't succeed, it returns the wait time needed
func (tb *tokenBucket) tryConsume(count int64, interval time.Duration) (bool, time.Duration) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.updateTokens()

	if tb.tokens >= count {
		tb.tokens -= count
		return true, 0
	}

	// Calculate the time needed to get enough tokens
	neededTokens := count - tb.tokens
	waitTime := time.Duration(float64(neededTokens) / tb.rate * float64(time.Second))

	// Limit wait time to the filling interval to avoid long waits
	if waitTime > interval {
		waitTime = interval
	}

	return false, waitTime
}

// consume blocks until the specified number of tokens can be consumed
func (tb *tokenBucket) consume(ctx context.Context, count int64, interval time.Duration) error {
	for {
		consumed, waitTime := tb.tryConsume(count, interval)
		if consumed {
			return nil
		}

		// Wait for more tokens or until context is canceled
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Continue and try again
		}
	}
}

// rateLimiterExtension implements a token bucket rate limiter and the Provider interface
type rateLimiterExtension struct {
	limiters        map[extensionlimiter.WeightKey]*tokenBucket
	fillingInterval time.Duration
	logger          *zap.Logger
}

// Ensure rateLimiterExtension implements the Provider interface
var _ extensionlimiter.Provider = (*rateLimiterExtension)(nil)

// newRateLimiter creates a new rate limiter extension with the given config
func newRateLimiter(_ context.Context, settings extension.Settings, cfg *Config) (*rateLimiterExtension, error) {
	limiters := make(map[extensionlimiter.WeightKey]*tokenBucket)

	for _, limit := range cfg.Limits {
		limiters[limit.Key] = newTokenBucket(limit.Rate, limit.BurstSize)
	}

	return &rateLimiterExtension{
		limiters:        limiters,
		fillingInterval: cfg.FillingInterval,
		logger:          settings.Logger,
	}, nil
}

// Start initializes the extension (no-op for this extension)
func (rl *rateLimiterExtension) Start(ctx context.Context, host component.Host) error {
	return nil
}

// Shutdown stops the extension (no-op for this extension)
func (rl *rateLimiterExtension) Shutdown(ctx context.Context) error {
	return nil
}

// RateLimiter returns a RateLimiter for the specified weight key or nil if not configured
func (rl *rateLimiterExtension) RateLimiter(key extensionlimiter.WeightKey) extensionlimiter.RateLimiter {
	limiter, ok := rl.limiters[key]
	if !ok {
		// No limiter configured for this weight key
		return nil
	}

	// Return a RateLimiterFunc that adapts our tokenBucket to the RateLimiter interface
	return extensionlimiter.RateLimiterFunc(func(ctx context.Context, weight uint64) error {
		return limiter.consume(ctx, int64(weight), rl.fillingInterval)
	})
}

// ResourceLimiter returns a ResourceLimiter for the specified weight key
func (rl *rateLimiterExtension) ResourceLimiter(key extensionlimiter.WeightKey) extensionlimiter.ResourceLimiter {
	rate := rl.RateLimiter(key)
	if rate == nil {
		return nil
	}
	return extensionlimiter.ResourceLimiterFunc(func(ctx context.Context, weight uint64) (extensionlimiter.ReleaseFunc, error) {
		return nil, rate.Limit(ctx, weight)
	})
}
