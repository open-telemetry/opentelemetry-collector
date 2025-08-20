// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

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
	}

	var resourceAttrs []config.AttributeNameValue
	for _, r := range res.Attributes() {
		resourceAttrs = append(resourceAttrs, config.AttributeNameValue{
			Name: string(r.Key), Value: r.Value.AsString(),
		})
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
					Attributes: resourceAttrs,
				},
			},
		),
	)
	if err != nil {
		return nil, err
	}
	return &sdk, nil
}
