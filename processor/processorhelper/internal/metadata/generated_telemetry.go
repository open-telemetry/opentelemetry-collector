// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"errors"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
)

// Deprecated: [v0.108.0] use LeveledMeter instead.
func Meter(settings component.TelemetrySettings) metric.Meter {
	return settings.MeterProvider.Meter("go.opentelemetry.io/collector/processor/processorhelper")
}

func LeveledMeter(settings component.TelemetrySettings, level configtelemetry.Level) metric.Meter {
	return settings.LeveledMeterProvider(level).Meter("go.opentelemetry.io/collector/processor/processorhelper")
}

func Tracer(settings component.TelemetrySettings) trace.Tracer {
	return settings.TracerProvider.Tracer("go.opentelemetry.io/collector/processor/processorhelper")
}

// TelemetryBuilder provides an interface for components to report telemetry
// as defined in metadata and user config.
type TelemetryBuilder struct {
	meter                         metric.Meter
	ProcessorAcceptedLogRecords   metric.Int64Counter
	ProcessorAcceptedMetricPoints metric.Int64Counter
	ProcessorAcceptedSpans        metric.Int64Counter
	ProcessorDroppedLogRecords    metric.Int64Counter
	ProcessorDroppedMetricPoints  metric.Int64Counter
	ProcessorDroppedSpans         metric.Int64Counter
	ProcessorIncomingLogRecords   metric.Int64Counter
	ProcessorIncomingMetricPoints metric.Int64Counter
	ProcessorIncomingSpans        metric.Int64Counter
	ProcessorOutgoingLogRecords   metric.Int64Counter
	ProcessorOutgoingMetricPoints metric.Int64Counter
	ProcessorOutgoingSpans        metric.Int64Counter
	ProcessorRefusedLogRecords    metric.Int64Counter
	ProcessorRefusedMetricPoints  metric.Int64Counter
	ProcessorRefusedSpans         metric.Int64Counter
	meters                        map[configtelemetry.Level]metric.Meter
}

// telemetryBuilderOption applies changes to default builder.
type telemetryBuilderOption func(*TelemetryBuilder)

// NewTelemetryBuilder provides a struct with methods to update all internal telemetry
// for a component
func NewTelemetryBuilder(settings component.TelemetrySettings, options ...telemetryBuilderOption) (*TelemetryBuilder, error) {
	builder := TelemetryBuilder{meters: map[configtelemetry.Level]metric.Meter{}}
	for _, op := range options {
		op(&builder)
	}
	builder.meters[configtelemetry.LevelBasic] = LeveledMeter(settings, configtelemetry.LevelBasic)
	var err, errs error
	builder.ProcessorAcceptedLogRecords, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_processor_accepted_log_records",
		metric.WithDescription("Number of log records successfully pushed into the next component in the pipeline."),
		metric.WithUnit("{records}"),
	)
	errs = errors.Join(errs, err)
	builder.ProcessorAcceptedMetricPoints, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_processor_accepted_metric_points",
		metric.WithDescription("Number of metric points successfully pushed into the next component in the pipeline."),
		metric.WithUnit("{datapoints}"),
	)
	errs = errors.Join(errs, err)
	builder.ProcessorAcceptedSpans, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_processor_accepted_spans",
		metric.WithDescription("Number of spans successfully pushed into the next component in the pipeline."),
		metric.WithUnit("{spans}"),
	)
	errs = errors.Join(errs, err)
	builder.ProcessorDroppedLogRecords, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_processor_dropped_log_records",
		metric.WithDescription("Number of log records that were dropped."),
		metric.WithUnit("{records}"),
	)
	errs = errors.Join(errs, err)
	builder.ProcessorDroppedMetricPoints, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_processor_dropped_metric_points",
		metric.WithDescription("Number of metric points that were dropped."),
		metric.WithUnit("{datapoints}"),
	)
	errs = errors.Join(errs, err)
	builder.ProcessorDroppedSpans, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_processor_dropped_spans",
		metric.WithDescription("Number of spans that were dropped."),
		metric.WithUnit("{spans}"),
	)
	errs = errors.Join(errs, err)
	builder.ProcessorIncomingLogRecords, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_processor_incoming_log_records",
		metric.WithDescription("Number of log records passed to the processor."),
		metric.WithUnit("{records}"),
	)
	errs = errors.Join(errs, err)
	builder.ProcessorIncomingMetricPoints, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_processor_incoming_metric_points",
		metric.WithDescription("Number of metric points passed to the processor."),
		metric.WithUnit("{datapoints}"),
	)
	errs = errors.Join(errs, err)
	builder.ProcessorIncomingSpans, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_processor_incoming_spans",
		metric.WithDescription("Number of spans passed to the processor."),
		metric.WithUnit("{spans}"),
	)
	errs = errors.Join(errs, err)
	builder.ProcessorOutgoingLogRecords, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_processor_outgoing_log_records",
		metric.WithDescription("Number of log records emitted from the processor."),
		metric.WithUnit("{records}"),
	)
	errs = errors.Join(errs, err)
	builder.ProcessorOutgoingMetricPoints, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_processor_outgoing_metric_points",
		metric.WithDescription("Number of metric points emitted from the processor."),
		metric.WithUnit("{datapoints}"),
	)
	errs = errors.Join(errs, err)
	builder.ProcessorOutgoingSpans, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_processor_outgoing_spans",
		metric.WithDescription("Number of spans emitted from the processor."),
		metric.WithUnit("{spans}"),
	)
	errs = errors.Join(errs, err)
	builder.ProcessorRefusedLogRecords, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_processor_refused_log_records",
		metric.WithDescription("Number of log records that were rejected by the next component in the pipeline."),
		metric.WithUnit("{records}"),
	)
	errs = errors.Join(errs, err)
	builder.ProcessorRefusedMetricPoints, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_processor_refused_metric_points",
		metric.WithDescription("Number of metric points that were rejected by the next component in the pipeline."),
		metric.WithUnit("{datapoints}"),
	)
	errs = errors.Join(errs, err)
	builder.ProcessorRefusedSpans, err = builder.meters[configtelemetry.LevelBasic].Int64Counter(
		"otelcol_processor_refused_spans",
		metric.WithDescription("Number of spans that were rejected by the next component in the pipeline."),
		metric.WithUnit("{spans}"),
	)
	errs = errors.Join(errs, err)
	return &builder, errs
}
