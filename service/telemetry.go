// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service // import "go.opentelemetry.io/collector/service"

import (
	"context"

	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/collector/service/internal/proctelemetry"
)

type meterProvider struct {
	*sdkmetric.MeterProvider
}

type meterProviderSettings struct {
	res               *resource.Resource
	asyncErrorChannel chan error

	OtelMetricViews       []sdkmetric.View
	OtelMetricReader      sdkmetric.Reader
	DisableProcessMetrics bool
}

func newMeterProvider(set meterProviderSettings, disableHighCardinality bool) (metric.MeterProvider, error) {
	mp := &meterProvider{}

	opts := []sdkmetric.Option{
		sdkmetric.WithReader(set.OtelMetricReader),
		sdkmetric.WithView(set.OtelMetricViews...),
	}

	var err error
	mp.MeterProvider, err = proctelemetry.InitOpenTelemetry(set.res, opts, disableHighCardinality)
	if err != nil {
		return nil, err
	}
	return mp, nil
}

// Shutdown the meter provider and all the associated resources.
// The type signature of this method matches that of the sdkmetric.MeterProvider.
func (mp *meterProvider) Shutdown(ctx context.Context) error {
	return nil
}
