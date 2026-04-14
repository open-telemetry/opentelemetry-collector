// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"

	otelconf "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	noopmetric "go.opentelemetry.io/otel/metric/noop"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/service/telemetry"
)

func createMeterProvider(
	ctx context.Context,
	set telemetry.MeterSettings,
	componentConfig component.Config,
) (telemetry.MeterProvider, error) {
	cfg := componentConfig.(*Config)
	if cfg.Metrics.Level == configtelemetry.LevelNone {
		set.Logger.Info("Internal metrics telemetry disabled")
		return noopMeterProvider{MeterProvider: noopmetric.NewMeterProvider()}, nil
	} else if cfg.Metrics.Views == nil && set.DefaultViews != nil {
		cfg.Metrics.Views = set.DefaultViews(cfg.Metrics.Level)
	}

	resourceConfig, err := createFixedResourceConfig(&cfg.Resource, set.Resource)
	if err != nil {
		return nil, err
	}

	mpConfig := cfg.Metrics.MeterProvider
	sdk, err := otelconf.NewSDK(otelconf.WithContext(ctx), otelconf.WithOpenTelemetryConfiguration(otelconf.OpenTelemetryConfiguration{
		Resource:      resourceConfig,
		MeterProvider: &mpConfig,
	}))
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
