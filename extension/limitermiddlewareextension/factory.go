// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limitermiddlewareextension // import "go.opentelemetry.io/collector/extension/limitermiddlewareextension"

//go:generate mdatagen metadata.yaml

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/limitermiddlewareextension/internal/metadata"
)

// NewFactory returns a new factory for the Limiter Middleware extension.
func NewFactory() extension.Factory {
	return extension.NewFactory(
		metadata.Type,
		createDefaultConfig,
		func(ctx context.Context, set extension.Settings, cfg component.Config) (
			extension.Extension,
			error,
		) {
			return newLimiterMiddleware(ctx, cfg.(*Config), set)
		},
		metadata.ExtensionStability)
}

// CreateDefaultConfig creates the default configuration for extension.
func createDefaultConfig() component.Config {
	return &Config{}
}
