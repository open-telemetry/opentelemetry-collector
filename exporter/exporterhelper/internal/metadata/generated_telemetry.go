// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
)

// Deprecated: [v0.108.0] use LeveledMeter instead.
func Meter(settings component.TelemetrySettings) metric.Meter {
	return settings.MeterProvider.Meter("go.opentelemetry.io/collector/exporter/exporterhelper")
}

func LeveledMeter(settings component.TelemetrySettings, level configtelemetry.Level) metric.Meter {
	return settings.LeveledMeterProvider(level).Meter("go.opentelemetry.io/collector/exporter/exporterhelper")
}

func Tracer(settings component.TelemetrySettings) trace.Tracer {
	return settings.TracerProvider.Tracer("go.opentelemetry.io/collector/exporter/exporterhelper")
}

// TelemetryBuilder provides an interface for components to report telemetry
// as defined in metadata and user config.
type TelemetryBuilder struct {
	meter                             metric.Meter
	ExporterEnqueueFailedLogRecords   metric.Int64Counter
	ExporterEnqueueFailedMetricPoints metric.Int64Counter
	ExporterEnqueueFailedSpans        metric.Int64Counter
	ExporterQueueCapacity             metric.Int64ObservableGauge
	ExporterQueueSize                 metric.Int64ObservableGauge
	ExporterSendFailedLogRecords      metric.Int64Counter
	ExporterSendFailedMetricPoints    metric.Int64Counter
	ExporterSendFailedSpans           metric.Int64Counter
	ExporterSentLogRecords            metric.Int64Counter
	ExporterSentMetricPoints          metric.Int64Counter
	ExporterSentSpans                 metric.Int64Counter
	meters                            map[configtelemetry.Level]metric.Meter
}

// telemetryBuilderOption applies changes to default builder.
type telemetryBuilderOption func(*TelemetryBuilder)

// InitExporterQueueCapacity configures the ExporterQueueCapacity metric.
func (builder *TelemetryBuilder) InitExporterQueueCapacity(cb func() int64, opts ...metric.ObserveOption) error {
	var err error
	builder.ExporterQueueCapacity, err = builder.meters[configtelemetry.LevelBasic].Int64ObservableGauge(
		"otelcol_exporter_queue_capacity",
		metric.WithDescription("Fixed capacity of the retry queue (in batches)"),
		metric.WithUnit("{batches}"),
	)
	if err != nil {
		return err
	}
	_, err = builder.meters[configtelemetry.LevelBasic].RegisterCallback(func(_ context.Context, o metric.Observer) error {
		o.ObserveInt64(builder.ExporterQueueCapacity, cb(), opts...)
		return nil
	}, builder.ExporterQueueCapacity)
	return err
}

// InitExporterQueueSize configures the ExporterQueueSize metric.
func (builder *TelemetryBuilder) InitExporterQueueSize(cb func() int64, opts ...metric.ObserveOption) error {
	var err error
	builder.ExporterQueueSize, err = builder.meters[configtelemetry.LevelBasic].Int64ObservableGauge(
		"otelcol_exporter_queue_size",
		metric.WithDescription("Current size of the retry queue (in batches)"),
		metric.WithUnit("{batches}"),
	)
	if err != nil {
		return err
	}
	_, err = builder.meters[configtelemetry.LevelBasic].RegisterCallback(func(_ context.Context, o metric.Observer) error {
		o.ObserveInt64(builder.ExporterQueueSize, cb(), opts...)
		return nil
	}, builder.ExporterQueueSize)
	return err
}

// NewTelemetryBuilder provides a struct with methods to update all internal telemetry
// for a component
func NewTelemetryBuilder(settings component.TelemetrySettings, options ...telemetryBuilderOption) (*TelemetryBuilder, error) {
	builder := TelemetryBuilder{meters: map[configtelemetry.Level]metric.Meter{}}
	for _, op := range options {
		op(&builder)
	}
	builder.meters[configtelemetry.LevelBasic] = LeveledMeter(settings, configtelemetry.LevelBasic)
	var err, errs error
	builder.ExporterEnqueueFailedLogRecords, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_exporter_enqueue_failed_log_records",
		metric.WithDescription("Number of log records failed to be added to the sending queue."),
		metric.WithUnit("{records}"),
	)
	errs = errors.Join(errs, err)
	builder.ExporterEnqueueFailedMetricPoints, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_exporter_enqueue_failed_metric_points",
		metric.WithDescription("Number of metric points failed to be added to the sending queue."),
		metric.WithUnit("{datapoints}"),
	)
	errs = errors.Join(errs, err)
	builder.ExporterEnqueueFailedSpans, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_exporter_enqueue_failed_spans",
		metric.WithDescription("Number of spans failed to be added to the sending queue."),
		metric.WithUnit("{spans}"),
	)
	errs = errors.Join(errs, err)
	builder.ExporterSendFailedLogRecords, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_exporter_send_failed_log_records",
		metric.WithDescription("Number of log records in failed attempts to send to destination."),
		metric.WithUnit("{records}"),
	)
	errs = errors.Join(errs, err)
	builder.ExporterSendFailedMetricPoints, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_exporter_send_failed_metric_points",
		metric.WithDescription("Number of metric points in failed attempts to send to destination."),
		metric.WithUnit("{datapoints}"),
	)
	errs = errors.Join(errs, err)
	builder.ExporterSendFailedSpans, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_exporter_send_failed_spans",
		metric.WithDescription("Number of spans in failed attempts to send to destination."),
		metric.WithUnit("{spans}"),
	)
	errs = errors.Join(errs, err)
	builder.ExporterSentLogRecords, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_exporter_sent_log_records",
		metric.WithDescription("Number of log record successfully sent to destination."),
		metric.WithUnit("{records}"),
	)
	errs = errors.Join(errs, err)
	builder.ExporterSentMetricPoints, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_exporter_sent_metric_points",
		metric.WithDescription("Number of metric points successfully sent to destination."),
		metric.WithUnit("{datapoints}"),
	)
	errs = errors.Join(errs, err)
	builder.ExporterSentSpans, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_exporter_sent_spans",
		metric.WithDescription("Number of spans successfully sent to destination."),
		metric.WithUnit("{spans}"),
	)
	errs = errors.Join(errs, err)
	return &builder, errs
}
