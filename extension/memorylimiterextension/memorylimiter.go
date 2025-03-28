// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiterextension // import "go.opentelemetry.io/collector/extension/memorylimiterextension"

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensionlimiter"
	"go.opentelemetry.io/collector/internal/memorylimiter"
)

type memoryLimiterExtension struct {
	memLimiter *memorylimiter.MemoryLimiter
}

var _ extensionlimiter.Provider = &memoryLimiterExtension{}

var ErrMemoryLimitExceeded = errors.New("memory limit exceeded")

// newMemoryLimiter returns a new memorylimiter extension.
func newMemoryLimiter(cfg *Config, logger *zap.Logger) (*memoryLimiterExtension, error) {
	ml, err := memorylimiter.NewMemoryLimiter(cfg, logger)
	if err != nil {
		return nil, err
	}

	return &memoryLimiterExtension{memLimiter: ml}, nil
}

// RateLimiter implements extensionlimiter.Provider.
func (ml *memoryLimiterExtension) RateLimiter(_ extensionlimiter.WeightKey) extensionlimiter.RateLimiter {
	return extensionlimiter.RateLimiterFunc(func(_ context.Context, _ uint64) error {
		if ml.memLimiter.MustRefuse() {
			return ErrMemoryLimitExceeded
		}
		return nil
	})
}

// ResourceLimiter implements extensionlimiter.Provider.
func (ml *memoryLimiterExtension) ResourceLimiter(key extensionlimiter.WeightKey) extensionlimiter.ResourceLimiter {
	return extensionlimiter.ResourceLimiterFunc(func(_ context.Context, _ uint64) (extensionlimiter.ReleaseFunc, error) {
		if ml.memLimiter.MustRefuse() {
			return nil, ErrMemoryLimitExceeded
		}
		return nil, nil
	})
}

func (ml *memoryLimiterExtension) Start(ctx context.Context, host component.Host) error {
	return ml.memLimiter.Start(ctx, host)
}

func (ml *memoryLimiterExtension) Shutdown(ctx context.Context) error {
	return ml.memLimiter.Shutdown(ctx)
}

// MustRefuse returns if the caller should deny because memory has
// reached its configured limits. This is the original API; it is not
// clear that there are any callers other than memorylimiterprocessor.
func (ml *memoryLimiterExtension) MustRefuse() bool {
	return ml.memLimiter.MustRefuse()
}
