// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute // import "go.opentelemetry.io/collector/internal/telemetry/componentattribute"

import (
	"context"
	"slices"

	"go.opentelemetry.io/otel/metric"
)

type float64Counter struct {
	metric.Float64Counter
	option metric.MeasurementOption
}

func (mwa meterWithAttributes) Float64Counter(name string, options ...metric.Float64CounterOption) (metric.Float64Counter, error) {
	inst, err := mwa.Meter.Float64Counter(name, options...)
	return float64Counter{
		Float64Counter: inst,
		option:         mwa.option,
	}, err
}

func (inst float64Counter) Add(ctx context.Context, incr float64, options ...metric.AddOption) {
	options = slices.Insert(options, 0, metric.AddOption(inst.option))
	inst.Float64Counter.Add(ctx, incr, options...)
}

type float64UpDownCounter struct {
	metric.Float64UpDownCounter
	option metric.MeasurementOption
}

func (mwa meterWithAttributes) Float64UpDownCounter(name string, options ...metric.Float64UpDownCounterOption) (metric.Float64UpDownCounter, error) {
	inst, err := mwa.Meter.Float64UpDownCounter(name, options...)
	return float64UpDownCounter{
		Float64UpDownCounter: inst,
		option:               mwa.option,
	}, err
}

func (inst float64UpDownCounter) Add(ctx context.Context, incr float64, options ...metric.AddOption) {
	options = slices.Insert(options, 0, metric.AddOption(inst.option))
	inst.Float64UpDownCounter.Add(ctx, incr, options...)
}

type float64Histogram struct {
	metric.Float64Histogram
	option metric.MeasurementOption
}

func (mwa meterWithAttributes) Float64Histogram(name string, options ...metric.Float64HistogramOption) (metric.Float64Histogram, error) {
	inst, err := mwa.Meter.Float64Histogram(name, options...)
	return float64Histogram{
		Float64Histogram: inst,
		option:           mwa.option,
	}, err
}

func (inst float64Histogram) Record(ctx context.Context, incr float64, options ...metric.RecordOption) {
	options = slices.Insert(options, 0, metric.RecordOption(inst.option))
	inst.Float64Histogram.Record(ctx, incr, options...)
}

type float64Gauge struct {
	metric.Float64Gauge
	option metric.MeasurementOption
}

func (mwa meterWithAttributes) Float64Gauge(name string, options ...metric.Float64GaugeOption) (metric.Float64Gauge, error) {
	inst, err := mwa.Meter.Float64Gauge(name, options...)
	return float64Gauge{
		Float64Gauge: inst,
		option:       mwa.option,
	}, err
}

func (inst float64Gauge) Record(ctx context.Context, incr float64, options ...metric.RecordOption) {
	options = slices.Insert(options, 0, metric.RecordOption(inst.option))
	inst.Float64Gauge.Record(ctx, incr, options...)
}

type float64Observer struct {
	metric.Float64Observer
	option metric.MeasurementOption
}

func (obs float64Observer) Observe(value float64, options ...metric.ObserveOption) {
	options = slices.Insert(options, 0, metric.ObserveOption(obs.option))
	obs.Float64Observer.Observe(value, options...)
}

func wrapFloat64Callback(cb metric.Float64Callback, option metric.MeasurementOption) metric.Float64Callback {
	return func(ctx context.Context, io metric.Float64Observer) error {
		return cb(ctx, float64Observer{Float64Observer: io, option: option})
	}
}

func (mwa meterWithAttributes) Float64ObservableCounter(name string, options ...metric.Float64ObservableCounterOption) (metric.Float64ObservableCounter, error) {
	cfg := metric.NewFloat64ObservableCounterConfig(options...)
	newOptions := []metric.Float64ObservableCounterOption{
		metric.WithUnit(cfg.Unit()),
		metric.WithDescription(cfg.Description()),
	}
	for _, cb := range cfg.Callbacks() {
		newOptions = append(newOptions, metric.WithFloat64Callback(wrapFloat64Callback(cb, mwa.option)))
	}
	return mwa.Meter.Float64ObservableCounter(name, newOptions...)
}

func (mwa meterWithAttributes) Float64ObservableUpDownCounter(name string, options ...metric.Float64ObservableUpDownCounterOption) (metric.Float64ObservableUpDownCounter, error) {
	cfg := metric.NewFloat64ObservableUpDownCounterConfig(options...)
	newOptions := []metric.Float64ObservableUpDownCounterOption{
		metric.WithUnit(cfg.Unit()),
		metric.WithDescription(cfg.Description()),
	}
	for _, cb := range cfg.Callbacks() {
		newOptions = append(newOptions, metric.WithFloat64Callback(wrapFloat64Callback(cb, mwa.option)))
	}
	return mwa.Meter.Float64ObservableUpDownCounter(name, newOptions...)
}

func (mwa meterWithAttributes) Float64ObservableGauge(name string, options ...metric.Float64ObservableGaugeOption) (metric.Float64ObservableGauge, error) {
	cfg := metric.NewFloat64ObservableGaugeConfig(options...)
	newOptions := []metric.Float64ObservableGaugeOption{
		metric.WithUnit(cfg.Unit()),
		metric.WithDescription(cfg.Description()),
	}
	for _, cb := range cfg.Callbacks() {
		newOptions = append(newOptions, metric.WithFloat64Callback(wrapFloat64Callback(cb, mwa.option)))
	}
	return mwa.Meter.Float64ObservableGauge(name, newOptions...)
}
