// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute // import "go.opentelemetry.io/collector/internal/telemetry/componentattribute"

import (
	"context"
	"slices"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type meterProviderWithAttributes struct {
	metric.MeterProvider
	attrs []attribute.KeyValue
}

// MeterProviderWithAttributes creates a MeterProvider with a new set of injected instrumentation scope attributes.
func MeterProviderWithAttributes(mp metric.MeterProvider, attrs attribute.Set) metric.MeterProvider {
	if mpwa, ok := mp.(meterProviderWithAttributes); ok {
		mp = mpwa.MeterProvider
	}
	return meterProviderWithAttributes{
		MeterProvider: mp,
		attrs:         attrs.ToSlice(),
	}
}

func (mpwa meterProviderWithAttributes) Meter(name string, opts ...metric.MeterOption) metric.Meter {
	conf := metric.NewMeterConfig(opts...)
	attrSet := conf.InstrumentationAttributes()
	// prepend our attributes so they can be overwritten
	newAttrs := append(slices.Clone(mpwa.attrs), attrSet.ToSlice()...)
	// append our attribute set option to overwrite the old one
	opts = append(opts, metric.WithInstrumentationAttributes(newAttrs...))
	return contextAttributesMeter{Meter: mpwa.MeterProvider.Meter(name, opts...)}
}

// contextAttributesMeter is a wrapper around metric.Meter that injects attributes
// from the context into measurements of synchronous instruments.
//
// Asynchronous instruments are passed through without modification, as they are
// expected to be called with a background context, and thus will not have any
// context attributes set.
type contextAttributesMeter struct {
	metric.Meter
}

func (m contextAttributesMeter) Int64Counter(name string, options ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	c, err := m.Meter.Int64Counter(name, options...)
	return contextAttributesInt64Counter{Int64Counter: c}, err
}

func (m contextAttributesMeter) Int64UpDownCounter(name string, options ...metric.Int64UpDownCounterOption) (metric.Int64UpDownCounter, error) {
	c, err := m.Meter.Int64UpDownCounter(name, options...)
	return contextAttributesInt64UpDownCounter{Int64UpDownCounter: c}, err
}

func (m contextAttributesMeter) Int64Histogram(name string, options ...metric.Int64HistogramOption) (metric.Int64Histogram, error) {
	h, err := m.Meter.Int64Histogram(name, options...)
	return contextAttributesInt64Histogram{Int64Histogram: h}, err
}

func (m contextAttributesMeter) Int64Gauge(name string, options ...metric.Int64GaugeOption) (metric.Int64Gauge, error) {
	g, err := m.Meter.Int64Gauge(name, options...)
	return contextAttributesInt64Gauge{Int64Gauge: g}, err
}

func (m contextAttributesMeter) Float64Counter(name string, options ...metric.Float64CounterOption) (metric.Float64Counter, error) {
	c, err := m.Meter.Float64Counter(name, options...)
	return contextAttributesFloat64Counter{Float64Counter: c}, err
}

func (m contextAttributesMeter) Float64UpDownCounter(name string, options ...metric.Float64UpDownCounterOption) (metric.Float64UpDownCounter, error) {
	c, err := m.Meter.Float64UpDownCounter(name, options...)
	return contextAttributesFloat64UpDownCounter{Float64UpDownCounter: c}, err
}

func (m contextAttributesMeter) Float64Gauge(name string, options ...metric.Float64GaugeOption) (metric.Float64Gauge, error) {
	g, err := m.Meter.Float64Gauge(name, options...)
	return contextAttributesFloat64Gauge{Float64Gauge: g}, err
}

func (m contextAttributesMeter) Float64Histogram(name string, options ...metric.Float64HistogramOption) (metric.Float64Histogram, error) {
	h, err := m.Meter.Float64Histogram(name, options...)
	return contextAttributesFloat64Histogram{Float64Histogram: h}, err
}

type contextAttributesInt64Counter struct {
	metric.Int64Counter
}

func (c contextAttributesInt64Counter) Add(ctx context.Context, value int64, opts ...metric.AddOption) {
	addWithContextAttributes(ctx, c.Int64Counter, value, opts...)
}

type contextAttributesInt64UpDownCounter struct {
	metric.Int64UpDownCounter
}

func (c contextAttributesInt64UpDownCounter) Add(ctx context.Context, value int64, opts ...metric.AddOption) {
	addWithContextAttributes(ctx, c.Int64UpDownCounter, value, opts...)
}

type contextAttributesInt64Histogram struct {
	metric.Int64Histogram
}

func (h contextAttributesInt64Histogram) Record(ctx context.Context, value int64, opts ...metric.RecordOption) {
	recordWithContextAttributes(ctx, h.Int64Histogram, value, opts...)
}

type contextAttributesInt64Gauge struct {
	metric.Int64Gauge
}

func (g contextAttributesInt64Gauge) Record(ctx context.Context, value int64, opts ...metric.RecordOption) {
	recordWithContextAttributes(ctx, g.Int64Gauge, value, opts...)
}

type contextAttributesFloat64Counter struct {
	metric.Float64Counter
}

func (c contextAttributesFloat64Counter) Add(ctx context.Context, value float64, opts ...metric.AddOption) {
	addWithContextAttributes(ctx, c.Float64Counter, value, opts...)
}

type contextAttributesFloat64UpDownCounter struct {
	metric.Float64UpDownCounter
}

func (c contextAttributesFloat64UpDownCounter) Add(ctx context.Context, value float64, opts ...metric.AddOption) {
	addWithContextAttributes(ctx, c.Float64UpDownCounter, value, opts...)
}

type contextAttributesFloat64Histogram struct {
	metric.Float64Histogram
}

func (h contextAttributesFloat64Histogram) Record(ctx context.Context, value float64, opts ...metric.RecordOption) {
	recordWithContextAttributes(ctx, h.Float64Histogram, value, opts...)
}

type contextAttributesFloat64Gauge struct {
	metric.Float64Gauge
}

func (g contextAttributesFloat64Gauge) Record(ctx context.Context, value float64, opts ...metric.RecordOption) {
	recordWithContextAttributes(ctx, g.Float64Gauge, value, opts...)
}

type adder[T int64 | float64] interface {
	Add(ctx context.Context, value T, opts ...metric.AddOption)
}

func addWithContextAttributes[T int64 | float64](ctx context.Context, a adder[T], v T, opts ...metric.AddOption) {
	if attrs, ok := attributesFromContext(ctx); ok {
		// Prepend the attributes from the context to the options,
		// so any callsite options passed to Add take precedence.
		var opt metric.AddOption = metric.WithAttributeSet(attrs)
		opts = slices.Insert(opts, 0, opt)
	}
	a.Add(ctx, v, opts...)
}

type recorder[T int64 | float64] interface {
	Record(ctx context.Context, value T, opts ...metric.RecordOption)
}

func recordWithContextAttributes[T int64 | float64](ctx context.Context, r recorder[T], v T, opts ...metric.RecordOption) {
	if attrs, ok := attributesFromContext(ctx); ok {
		// Prepend the attributes from the context to the options,
		// so any callsite options passed to Record take precedence.
		var opt metric.RecordOption = metric.WithAttributeSet(attrs)
		opts = slices.Insert(opts, 0, opt)
	}
	r.Record(ctx, v, opts...)
}
