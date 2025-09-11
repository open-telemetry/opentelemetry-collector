// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"

	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/service/telemetry"
)

func createMeterProvider(
	ctx context.Context,
	set telemetry.Settings,
	componentConfig component.Config,
	logger *zap.Logger,
) (telemetry.MeterProvider, error) {
	cfg := componentConfig.(*Config)
	if cfg.Metrics.Level == configtelemetry.LevelNone {
		logger.Info("Internal metrics telemetry disabled")
		return noopMeterProvider{MeterProvider: noopmetric.NewMeterProvider()}, nil
	} else if cfg.Metrics.Views == nil && set.DefaultViews != nil {
		cfg.Metrics.Views = set.DefaultViews(cfg.Metrics.Level)
	}

	res := newResource(set, cfg)
	mpConfig := cfg.Metrics.MeterProvider
	sdk, err := newSDK(ctx, res, config.OpenTelemetryConfiguration{
		MeterProvider: &mpConfig,
	})
	if err != nil {
		return nil, err
	}
	return sdk.MeterProvider().(telemetry.MeterProvider), nil
}

type noopMeterProvider struct {
	noopmetric.MeterProvider
}

func (noopMeterProvider) Shutdown(context.Context) error {
	return nil
}
