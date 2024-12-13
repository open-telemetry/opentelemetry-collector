// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
)

func Meter(settings component.TelemetrySettings) metric.Meter {
	return settings.MeterProvider.Meter("go.opentelemetry.io/collector/internal/receiver/samplereceiver")
}

func Tracer(settings component.TelemetrySettings) trace.Tracer {
	return settings.TracerProvider.Tracer("go.opentelemetry.io/collector/internal/receiver/samplereceiver")
}

// TelemetryBuilder provides an interface for components to report telemetry
// as defined in metadata and user config.
type TelemetryBuilder struct {
	meter                                metric.Meter
	BatchSizeTriggerSend                 metric.Int64Counter
	ProcessRuntimeTotalAllocBytes        metric.Int64ObservableCounter
	observeProcessRuntimeTotalAllocBytes func(context.Context, metric.Observer) error
	QueueLength                          metric.Int64ObservableGauge
	RequestDuration                      metric.Float64Histogram
}

// TelemetryBuilderOption applies changes to default builder.
type TelemetryBuilderOption interface {
	apply(*TelemetryBuilder)
}

type telemetryBuilderOptionFunc func(mb *TelemetryBuilder)

func (tbof telemetryBuilderOptionFunc) apply(mb *TelemetryBuilder) {
	tbof(mb)
}

// WithProcessRuntimeTotalAllocBytesCallback sets callback for observable ProcessRuntimeTotalAllocBytes metric.
func WithProcessRuntimeTotalAllocBytesCallback(cb func() int64, opts ...metric.ObserveOption) TelemetryBuilderOption {
	return telemetryBuilderOptionFunc(func(builder *TelemetryBuilder) {
		builder.observeProcessRuntimeTotalAllocBytes = func(_ context.Context, o metric.Observer) error {
			o.ObserveInt64(builder.ProcessRuntimeTotalAllocBytes, cb(), opts...)
			return nil
		}
	})
}

// InitQueueLength configures the QueueLength metric.
func (builder *TelemetryBuilder) InitQueueLength(cb func() int64, opts ...metric.ObserveOption) (metric.Registration, error) {
	var err error
	builder.QueueLength, err = builder.meter.Int64ObservableGauge(
		"otelcol_queue_length",
		metric.WithDescription("This metric is optional and therefore not initialized in NewTelemetryBuilder."),
		metric.WithUnit("{items}"),
	)
	if err != nil {
		return nil, err
	}
	reg, err := builder.meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		o.ObserveInt64(builder.QueueLength, cb(), opts...)
		return nil
	}, builder.QueueLength)
	return reg, err
}

// NewTelemetryBuilder provides a struct with methods to update all internal telemetry
// for a component
func NewTelemetryBuilder(settings component.TelemetrySettings, options ...TelemetryBuilderOption) (*TelemetryBuilder, error) {
	builder := TelemetryBuilder{}
	for _, op := range options {
		op.apply(&builder)
	}
	builder.meter = Meter(settings)
	var err, errs error
	builder.BatchSizeTriggerSend, err = getLeveledMeter(builder.meter, configtelemetry.LevelBasic, settings.MetricsLevel).Int64Counter(
		"otelcol_batch_size_trigger_send",
		metric.WithDescription("Number of times the batch was sent due to a size trigger [deprecated since v0.110.0]"),
		metric.WithUnit("{times}"),
	)
	errs = errors.Join(errs, err)
	builder.ProcessRuntimeTotalAllocBytes, err = getLeveledMeter(builder.meter, configtelemetry.LevelBasic, settings.MetricsLevel).Int64ObservableCounter(
		"otelcol_process_runtime_total_alloc_bytes",
		metric.WithDescription("Cumulative bytes allocated for heap objects (see 'go doc runtime.MemStats.TotalAlloc')"),
		metric.WithUnit("By"),
	)
	errs = errors.Join(errs, err)
	_, err = getLeveledMeter(builder.meter, configtelemetry.LevelBasic, settings.MetricsLevel).RegisterCallback(builder.observeProcessRuntimeTotalAllocBytes, builder.ProcessRuntimeTotalAllocBytes)
	errs = errors.Join(errs, err)
	builder.RequestDuration, err = getLeveledMeter(builder.meter, configtelemetry.LevelBasic, settings.MetricsLevel).Float64Histogram(
		"otelcol_request_duration",
		metric.WithDescription("Duration of request [alpha]"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries([]float64{1, 10, 100}...),
	)
	errs = errors.Join(errs, err)
	return &builder, errs
}

func getLeveledMeter(meter metric.Meter, cfgLevel, srvLevel configtelemetry.Level) metric.Meter {
	if cfgLevel <= srvLevel {
		return meter
	}
	return noop.Meter{}
}
