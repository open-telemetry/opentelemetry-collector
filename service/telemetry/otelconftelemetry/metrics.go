// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"

	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"

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
	} else if cfg.Metrics.Views == nil && set.DefaultDroppedInstruments != nil {
		selectors := set.DefaultDroppedInstruments(cfg.Metrics.Level)
		cfg.Metrics.Views = buildViewsFromSelectors(selectors)
	}

	attrs := pcommonAttrsToOTelAttrs(set.Resource)
	res := sdkresource.NewWithAttributes("", attrs...)
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

// buildViewsFromSelectors converts InstrumentSelectors into views.
//
// Each selector is converted to a drop view that matches the specified
// meter and instrument name patterns.
func buildViewsFromSelectors(selectors []telemetry.InstrumentSelector) []config.View {
	views := make([]config.View, len(selectors))
	for i, selector := range selectors {
		viewSelector := &config.ViewSelector{}
		if selector.MeterName != "" {
			viewSelector.MeterName = ptr(selector.MeterName)
		}
		if selector.InstrumentName != "" {
			viewSelector.InstrumentName = ptr(selector.InstrumentName)
		}
		views[i] = config.View{
			Selector: viewSelector,
			Stream: &config.ViewStream{
				Aggregation: &config.ViewStreamAggregation{
					Drop: config.ViewStreamAggregationDrop{},
				},
			},
		}
	}
	return views
}
