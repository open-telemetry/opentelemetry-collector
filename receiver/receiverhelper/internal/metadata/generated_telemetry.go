// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"errors"
	"sync"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
)

func Meter(settings component.TelemetrySettings) metric.Meter {
	return settings.MeterProvider.Meter("go.opentelemetry.io/collector/receiver/receiverhelper")
}

func Tracer(settings component.TelemetrySettings) trace.Tracer {
	return settings.TracerProvider.Tracer("go.opentelemetry.io/collector/receiver/receiverhelper")
}

// TelemetryBuilder provides an interface for components to report telemetry
// as defined in metadata and user config.
type TelemetryBuilder struct {
	meter                              metric.Meter
	mu                                 sync.Mutex
	registrations                      []metric.Registration
	ReceiverAcceptedLogRecords         metric.Int64Counter
	ReceiverAcceptedMetricPoints       metric.Int64Counter
	ReceiverAcceptedSpans              metric.Int64Counter
	ReceiverInternalErrorsLogRecords   metric.Int64Counter
	ReceiverInternalErrorsMetricPoints metric.Int64Counter
	ReceiverInternalErrorsSpans        metric.Int64Counter
	ReceiverRefusedLogRecords          metric.Int64Counter
	ReceiverRefusedMetricPoints        metric.Int64Counter
	ReceiverRefusedSpans               metric.Int64Counter
	ReceiverRequests                   metric.Int64Counter
}

// TelemetryBuilderOption applies changes to default builder.
type TelemetryBuilderOption interface {
	apply(*TelemetryBuilder)
}

type telemetryBuilderOptionFunc func(mb *TelemetryBuilder)

func (tbof telemetryBuilderOptionFunc) apply(mb *TelemetryBuilder) {
	tbof(mb)
}

// Shutdown unregister all registered callbacks for async instruments.
func (builder *TelemetryBuilder) Shutdown() {
	builder.mu.Lock()
	defer builder.mu.Unlock()
	for _, reg := range builder.registrations {
		reg.Unregister()
	}
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
	builder.ReceiverRequests, err = builder.meter.Int64Counter(
		"otelcol_receiver_requests",
		metric.WithDescription("The number of requests received by the receiver"),
		metric.WithUnit("{requests}"),
	)
	errs = errors.Join(errs, err)
	builder.ReceiverAcceptedLogRecords, err = builder.meter.Int64Counter(
		"otelcol_receiver_accepted_log_records",
		metric.WithDescription("Number of log records successfully pushed into the pipeline. [alpha]"),
		metric.WithUnit("{records}"),
	)
	errs = errors.Join(errs, err)
	builder.ReceiverAcceptedMetricPoints, err = builder.meter.Int64Counter(
		"otelcol_receiver_accepted_metric_points",
		metric.WithDescription("Number of metric points successfully pushed into the pipeline. [alpha]"),
		metric.WithUnit("{datapoints}"),
	)
	errs = errors.Join(errs, err)
	builder.ReceiverAcceptedSpans, err = builder.meter.Int64Counter(
		"otelcol_receiver_accepted_spans",
		metric.WithDescription("Number of spans successfully pushed into the pipeline. [alpha]"),
		metric.WithUnit("{spans}"),
	)
	errs = errors.Join(errs, err)
	builder.ReceiverInternalErrorsLogRecords, err = builder.meter.Int64Counter(
		"otelcol_receiver_internal_errors_log_records",
		metric.WithDescription("Number of log records that could not be processed due to internal errors (e.g. parsing, validation). [alpha]"),
		metric.WithUnit("{records}"),
	)
	errs = errors.Join(errs, err)
	builder.ReceiverInternalErrorsMetricPoints, err = builder.meter.Int64Counter(
		"otelcol_receiver_internal_errors_metric_points",
		metric.WithDescription("Number of metric points that could not be processed due to internal errors (e.g. parsing, validation). [alpha]"),
		metric.WithUnit("{datapoints}"),
	)
	errs = errors.Join(errs, err)
	builder.ReceiverInternalErrorsSpans, err = builder.meter.Int64Counter(
		"otelcol_receiver_internal_errors_spans",
		metric.WithDescription("Number of spans that could not be processed due to internal errors (e.g. parsing, validation). [alpha]"),
		metric.WithUnit("{spans}"),
	)
	errs = errors.Join(errs, err)
	builder.ReceiverRefusedLogRecords, err = builder.meter.Int64Counter(
		"otelcol_receiver_refused_log_records",
		metric.WithDescription("Number of log records that could not be pushed into the pipeline due to downstream errors (e.g. from nextConsumer.ConsumeX). [alpha]"),
		metric.WithUnit("{records}"),
	)
	errs = errors.Join(errs, err)
	builder.ReceiverRefusedMetricPoints, err = builder.meter.Int64Counter(
		"otelcol_receiver_refused_metric_points",
		metric.WithDescription("Number of metric points that could not be pushed into the pipeline due to downstream errors (e.g. from nextConsumer.ConsumeX). [alpha]"),
		metric.WithUnit("{datapoints}"),
	)
	errs = errors.Join(errs, err)
	builder.ReceiverRefusedSpans, err = builder.meter.Int64Counter(
		"otelcol_receiver_refused_spans",
		metric.WithDescription("Number of spans that could not be pushed into the pipeline due to downstream errors (e.g. from nextConsumer.ConsumeX). [alpha]"),
		metric.WithUnit("{spans}"),
	)
	errs = errors.Join(errs, err)
	return &builder, errs
}
