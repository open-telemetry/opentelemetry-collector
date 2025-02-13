package componentattribute

import (
	"context"
	"slices"

	"go.opentelemetry.io/otel/metric"
)

type int64Counter struct {
	metric.Int64Counter
	option metric.MeasurementOption
}

func (mwa meterWithAttributes) Int64Counter(name string, options ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	inst, err := mwa.Meter.Int64Counter(name, options...)
	return int64Counter{
		Int64Counter: inst,
		option:       mwa.option,
	}, err
}
func (inst int64Counter) Add(ctx context.Context, incr int64, options ...metric.AddOption) {
	options = slices.Insert(options, 0, metric.AddOption(inst.option))
	inst.Int64Counter.Add(ctx, incr, options...)
}

type int64UpDownCounter struct {
	metric.Int64UpDownCounter
	option metric.MeasurementOption
}

func (mwa meterWithAttributes) Int64UpDownCounter(name string, options ...metric.Int64UpDownCounterOption) (metric.Int64UpDownCounter, error) {
	inst, err := mwa.Meter.Int64UpDownCounter(name, options...)
	return int64UpDownCounter{
		Int64UpDownCounter: inst,
		option:             mwa.option,
	}, err
}
func (inst int64UpDownCounter) Add(ctx context.Context, incr int64, options ...metric.AddOption) {
	options = slices.Insert(options, 0, metric.AddOption(inst.option))
	inst.Int64UpDownCounter.Add(ctx, incr, options...)
}

type int64Histogram struct {
	metric.Int64Histogram
	option metric.MeasurementOption
}

func (mwa meterWithAttributes) Int64Histogram(name string, options ...metric.Int64HistogramOption) (metric.Int64Histogram, error) {
	inst, err := mwa.Meter.Int64Histogram(name, options...)
	return int64Histogram{
		Int64Histogram: inst,
		option:         mwa.option,
	}, err
}
func (inst int64Histogram) Record(ctx context.Context, incr int64, options ...metric.RecordOption) {
	options = slices.Insert(options, 0, metric.RecordOption(inst.option))
	inst.Int64Histogram.Record(ctx, incr, options...)
}

type int64Gauge struct {
	metric.Int64Gauge
	option metric.MeasurementOption
}

func (mwa meterWithAttributes) Int64Gauge(name string, options ...metric.Int64GaugeOption) (metric.Int64Gauge, error) {
	inst, err := mwa.Meter.Int64Gauge(name, options...)
	return int64Gauge{
		Int64Gauge: inst,
		option:     mwa.option,
	}, err
}
func (inst int64Gauge) Record(ctx context.Context, incr int64, options ...metric.RecordOption) {
	options = slices.Insert(options, 0, metric.RecordOption(inst.option))
	inst.Int64Gauge.Record(ctx, incr, options...)
}

type int64Observer struct {
	metric.Int64Observer
	option metric.MeasurementOption
}

func (obs int64Observer) Observe(value int64, options ...metric.ObserveOption) {
	options = slices.Insert(options, 0, metric.ObserveOption(obs.option))
	obs.Int64Observer.Observe(value, options...)
}

func wrapInt64Callback(cb metric.Int64Callback, option metric.MeasurementOption) metric.Int64Callback {
	return func(ctx context.Context, io metric.Int64Observer) error {
		return cb(ctx, int64Observer{Int64Observer: io, option: option})
	}
}

func (mwa meterWithAttributes) Int64ObservableCounter(name string, options ...metric.Int64ObservableCounterOption) (metric.Int64ObservableCounter, error) {
	cfg := metric.NewInt64ObservableCounterConfig(options...)
	newOptions := []metric.Int64ObservableCounterOption{
		metric.WithUnit(cfg.Unit()),
		metric.WithDescription(cfg.Description()),
	}
	for _, cb := range cfg.Callbacks() {
		newOptions = append(newOptions, metric.WithInt64Callback(wrapInt64Callback(cb, mwa.option)))
	}
	return mwa.Meter.Int64ObservableCounter(name, newOptions...)
}

func (mwa meterWithAttributes) Int64ObservableUpDownCounter(name string, options ...metric.Int64ObservableUpDownCounterOption) (metric.Int64ObservableUpDownCounter, error) {
	cfg := metric.NewInt64ObservableUpDownCounterConfig(options...)
	newOptions := []metric.Int64ObservableUpDownCounterOption{
		metric.WithUnit(cfg.Unit()),
		metric.WithDescription(cfg.Description()),
	}
	for _, cb := range cfg.Callbacks() {
		newOptions = append(newOptions, metric.WithInt64Callback(wrapInt64Callback(cb, mwa.option)))
	}
	return mwa.Meter.Int64ObservableUpDownCounter(name, newOptions...)
}

func (mwa meterWithAttributes) Int64ObservableGauge(name string, options ...metric.Int64ObservableGaugeOption) (metric.Int64ObservableGauge, error) {
	cfg := metric.NewInt64ObservableGaugeConfig(options...)
	newOptions := []metric.Int64ObservableGaugeOption{
		metric.WithUnit(cfg.Unit()),
		metric.WithDescription(cfg.Description()),
	}
	for _, cb := range cfg.Callbacks() {
		newOptions = append(newOptions, metric.WithInt64Callback(wrapInt64Callback(cb, mwa.option)))
	}
	return mwa.Meter.Int64ObservableGauge(name, newOptions...)
}
