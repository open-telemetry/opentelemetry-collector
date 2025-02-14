// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute

import (
	"context"
	"slices"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type meterProviderWithAttributes struct {
	metric.MeterProvider
	option metric.MeasurementOption
}

func MeterProviderWithAttributes(mp metric.MeterProvider, attrs attribute.Set) metric.MeterProvider {
	if mpwa, ok := mp.(meterProviderWithAttributes); ok {
		mp = mpwa.MeterProvider
	}
	return meterProviderWithAttributes{
		MeterProvider: mp,
		option:        metric.WithAttributeSet(attrs),
	}
}

type meterWithAttributes struct {
	metric.Meter
	option metric.MeasurementOption
}

func (mpwa meterProviderWithAttributes) Meter(name string, opts ...metric.MeterOption) metric.Meter {
	opts = append(opts, metric.WithInstrumentationAttributes())
	return meterWithAttributes{
		Meter:  mpwa.MeterProvider.Meter(name, opts...),
		option: mpwa.option,
	}
}

type observerWithAttributes struct {
	metric.Observer
	option metric.MeasurementOption
}

func (obs observerWithAttributes) ObserveInt64(obsrv metric.Int64Observable, value int64, options ...metric.ObserveOption) {
	options = slices.Insert(options, 0, metric.ObserveOption(obs.option))
	obs.Observer.ObserveInt64(obsrv, value, options...)
}

func (obs observerWithAttributes) ObserveFloat64(obsrv metric.Float64Observable, value float64, options ...metric.ObserveOption) {
	options = slices.Insert(options, 0, metric.ObserveOption(obs.option))
	obs.Observer.ObserveFloat64(obsrv, value, options...)
}

func (mwa meterWithAttributes) RegisterCallback(f metric.Callback, instruments ...metric.Observable) (metric.Registration, error) {
	return mwa.Meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		return f(ctx, observerWithAttributes{Observer: o, option: mwa.option})
	}, instruments...)
}
