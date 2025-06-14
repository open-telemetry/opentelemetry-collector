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

var (
	ErrMustRefuse = errors.New("system is near memory limit")
)

type memoryLimiterExtension struct {
	memLimiter *memorylimiter.MemoryLimiter
}

var _ extensionlimiter.BaseLimiterProvider = &memoryLimiterExtension{}

// newMemoryLimiter returns a new memorylimiter extension.
func newMemoryLimiter(cfg *Config, logger *zap.Logger) (*memoryLimiterExtension, error) {
	ml, err := memorylimiter.NewMemoryLimiter(cfg, logger)
	if err != nil {
		return nil, err
	}

	return &memoryLimiterExtension{memLimiter: ml}, nil
}

func (ml *memoryLimiterExtension) Start(ctx context.Context, host component.Host) error {
	return ml.memLimiter.Start(ctx, host)
}

func (ml *memoryLimiterExtension) Shutdown(ctx context.Context) error {
	return ml.memLimiter.Shutdown(ctx)
}

// GetBaseLimiter implements extensionlimiter.BaseLimiterProvider.
func (ml *memoryLimiterExtension) GetBaseLimiter(
	opts ...extensionlimiter.Option,
) (extensionlimiter.BaseLimiter, error) {
	return extensionlimiter.MustDenyFunc(func(_ context.Context) error {
		if ml.MustRefuse() {
			return ErrMustRefuse
		}
		return nil
	}), nil
}

// MustRefuse returns if the caller should deny because memory has reached it's configured limits
func (ml *memoryLimiterExtension) MustRefuse() bool {
	return ml.memLimiter.MustRefuse()
}
