// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ratelimiterextension // import "go.opentelemetry.io/collector/extension/ratelimiterextension"

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionlimiter"
	"go.opentelemetry.io/collector/extension/ratelimiterextension/internal/metadata"
)

//go:generate mdatagen metadata.yaml

// NewFactory creates a factory for the rate limiter extension.
func NewFactory() extension.Factory {
	return extension.NewFactory(
		metadata.Type,
		createDefaultConfig,
		func(ctx context.Context, set extension.Settings, cfg component.Config) (
			extension.Extension,
			error,
		) {
			return newRateLimiter(ctx, set, cfg.(*Config))
		},
		metadata.ExtensionStability,
	)
}

// createDefaultConfig creates a relatively large default config.
func createDefaultConfig() component.Config {
	return &Config{
		Limits: []LimitConfig{
			{
				Key:       extensionlimiter.WeightKeyNetworkBytes,
				Rate:      10485760.0, // 10 MiB per second
				BurstSize: 52428800,   // Allow bursts of up to 50 MiB
			},
		},
		FillingInterval: 100 * time.Millisecond, // Check for tokens every 100ms
	}
}
