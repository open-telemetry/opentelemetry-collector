// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"

	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"go.opentelemetry.io/collector/config/configtelemetry"
)

// NewSDK creates a new OpenTelemetry SDK with the provided configuration.
func NewSDK(ctx context.Context, cfg *Config, res *resource.Resource) (*config.SDK, error) {
	mpConfig := cfg.Metrics.MeterProvider
	if cfg.Metrics.Level == configtelemetry.LevelNone {
		mpConfig.Readers = nil
	} else if mpConfig.Views == nil {
		mpConfig.Views = configureViews(cfg.Metrics.Level)
	}

	sdk, err := config.NewSDK(
		config.WithContext(ctx),
		config.WithOpenTelemetryConfiguration(
			config.OpenTelemetryConfiguration{
				LoggerProvider: &config.LoggerProvider{
					Processors: cfg.Logs.Processors,
				},
				MeterProvider:  &mpConfig,
				TracerProvider: &cfg.Traces.TracerProvider,
				Resource: &config.Resource{
					SchemaUrl:  ptr(semconv.SchemaURL),
					Attributes: attributes(res, cfg),
				},
			},
		),
	)
	if err != nil {
		return nil, err
	}
	return &sdk, nil
}
