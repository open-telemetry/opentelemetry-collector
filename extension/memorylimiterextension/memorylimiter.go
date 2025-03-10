// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiterextension // import "go.opentelemetry.io/collector/extension/memorylimiterextension"

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/xextension/limit"
	"go.opentelemetry.io/collector/internal/memorylimiter"
)

type memoryLimiterExtension struct {
	memLimiter *memorylimiter.MemoryLimiter
}

var _ limit.Extension = &memoryLimiterExtension{}

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

// MustRefuse returns if the caller should deny because memory has reached it's configured limits
//
// It's not clear that this is used anywhere, but as a legacy exported
// function some component could observe it disappear, so it has to stay.
func (ml *memoryLimiterExtension) MustRefuse() bool {
	return ml.memLimiter.MustRefuse()
}

// client implements Client. Note that the component Kind and ID are
// not used, so this struct is identical to *memoryLimiterExtension.
type client struct {
	ml *memoryLimiterExtension
}

// GetClient implements limit.Extension.
func (ml *memoryLimiterExtension) GetClient(_ context.Context, _ component.Kind, _ component.ID) (limit.Client, error) {
	return &client{ml}, nil
}

// Acquire implements limit.Client.
func (c *client) Acquire(_ context.Context, _ uint64) (limit.ReleaseFunc, error) {
	if c.ml.memLimiter.MustRefuse() {
		return nil, memorylimiter.ErrDataRefused
	}
	return func() {}, nil
}
