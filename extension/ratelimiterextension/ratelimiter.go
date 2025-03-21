package ratelimiterextension

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionlimiter"
)

type rateLimiterExtension struct {
}

var _ extensionlimiter.RateLimiter = (*rateLimiterExtension)(nil)

func newRateLimiter(_ context.Context, _ extension.Settings, _ *Config) (*rateLimiterExtension, error) {
	return &rateLimiterExtension{}, nil
}

func (rl *rateLimiterExtension) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (rl *rateLimiterExtension) Shutdown(ctx context.Context) error {
	return nil
}

func (rl *rateLimiterExtension) Limit(ctx context.Context, weights []extensionlimiter.Weight) {
	return
}
