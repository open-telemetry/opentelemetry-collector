package ratelimiterextension

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/ratelimiterextension/internal/metadata"
)

//go:generate mdatagen metadata.yaml

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

func createDefaultConfig() component.Config {
	return &Config{}
}
