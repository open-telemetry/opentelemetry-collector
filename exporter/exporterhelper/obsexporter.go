// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

const (
	exporterScope = obsmetrics.Scope + obsmetrics.NameSep + obsmetrics.ExporterKey
)

// ObsReport is a helper to add observability to an exporter.
type ObsReport struct {
	level          configtelemetry.Level
	spanNamePrefix string
	tracer         trace.Tracer
	logger         *zap.Logger

	otelAttrs                   []attribute.KeyValue
	sentSpans                   metric.Int64Counter
	failedToSendSpans           metric.Int64Counter
	failedToEnqueueSpans        metric.Int64Counter
	sentMetricPoints            metric.Int64Counter
	failedToSendMetricPoints    metric.Int64Counter
	failedToEnqueueMetricPoints metric.Int64Counter
	sentLogRecords              metric.Int64Counter
	failedToSendLogRecords      metric.Int64Counter
	failedToEnqueueLogRecords   metric.Int64Counter
}

// ObsReportSettings are settings for creating an ObsReport.
type ObsReportSettings struct {
	ExporterID             component.ID
	ExporterCreateSettings exporter.CreateSettings
}

// NewObsReport creates a new Exporter.
func NewObsReport(cfg ObsReportSettings) (*ObsReport, error) {
	return newExporter(cfg)
}

func newExporter(cfg ObsReportSettings) (*ObsReport, error) {
	exp := &ObsReport{
		level:          cfg.ExporterCreateSettings.TelemetrySettings.MetricsLevel,
		spanNamePrefix: obsmetrics.ExporterPrefix + cfg.ExporterID.String(),
		tracer:         cfg.ExporterCreateSettings.TracerProvider.Tracer(cfg.ExporterID.String()),
		logger:         cfg.ExporterCreateSettings.Logger,

		otelAttrs: []attribute.KeyValue{
			attribute.String(obsmetrics.ExporterKey, cfg.ExporterID.String()),
		},
	}

	if err := exp.createOtelMetrics(cfg); err != nil {
		return nil, err
	}

	return exp, nil
}

func (or *ObsReport) createOtelMetrics(cfg ObsReportSettings) error {
	meter := cfg.ExporterCreateSettings.MeterProvider.Meter(exporterScope)

	var errors, err error

	or.sentSpans, err = meter.Int64Counter(
		obsmetrics.ExporterPrefix+obsmetrics.SentSpansKey,
		metric.WithDescription("Number of spans successfully sent to destination."),
		metric.WithUnit("1"))
	errors = multierr.Append(errors, err)

	or.failedToSendSpans, err = meter.Int64Counter(
		obsmetrics.ExporterPrefix+obsmetrics.FailedToSendSpansKey,
		metric.WithDescription("Number of spans in failed attempts to send to destination."),
		metric.WithUnit("1"))
	errors = multierr.Append(errors, err)

	or.failedToEnqueueSpans, err = meter.Int64Counter(
		obsmetrics.ExporterPrefix+obsmetrics.FailedToEnqueueSpansKey,
		metric.WithDescription("Number of spans failed to be added to the sending queue."),
		metric.WithUnit("1"))
	errors = multierr.Append(errors, err)

	or.sentMetricPoints, err = meter.Int64Counter(
		obsmetrics.ExporterPrefix+obsmetrics.SentMetricPointsKey,
		metric.WithDescription("Number of metric points successfully sent to destination."),
		metric.WithUnit("1"))
	errors = multierr.Append(errors, err)

	or.failedToSendMetricPoints, err = meter.Int64Counter(
		obsmetrics.ExporterPrefix+obsmetrics.FailedToSendMetricPointsKey,
		metric.WithDescription("Number of metric points in failed attempts to send to destination."),
		metric.WithUnit("1"))
	errors = multierr.Append(errors, err)

	or.failedToEnqueueMetricPoints, err = meter.Int64Counter(
		obsmetrics.ExporterPrefix+obsmetrics.FailedToEnqueueMetricPointsKey,
		metric.WithDescription("Number of metric points failed to be added to the sending queue."),
		metric.WithUnit("1"))
	errors = multierr.Append(errors, err)

	or.sentLogRecords, err = meter.Int64Counter(
		obsmetrics.ExporterPrefix+obsmetrics.SentLogRecordsKey,
		metric.WithDescription("Number of log record successfully sent to destination."),
		metric.WithUnit("1"))
	errors = multierr.Append(errors, err)

	or.failedToSendLogRecords, err = meter.Int64Counter(
		obsmetrics.ExporterPrefix+obsmetrics.FailedToSendLogRecordsKey,
		metric.WithDescription("Number of log records in failed attempts to send to destination."),
		metric.WithUnit("1"))
	errors = multierr.Append(errors, err)

	or.failedToEnqueueLogRecords, err = meter.Int64Counter(
		obsmetrics.ExporterPrefix+obsmetrics.FailedToEnqueueLogRecordsKey,
		metric.WithDescription("Number of log records failed to be added to the sending queue."),
		metric.WithUnit("1"))
	errors = multierr.Append(errors, err)

	return errors
}

// StartTracesOp is called at the start of an Export operation.
// The returned context should be used in other calls to the Exporter functions
// dealing with the same export operation.
func (or *ObsReport) StartTracesOp(ctx context.Context) context.Context {
	return or.startOp(ctx, obsmetrics.ExportTraceDataOperationSuffix)
}

// EndTracesOp completes the export operation that was started with StartTracesOp.
func (or *ObsReport) EndTracesOp(ctx context.Context, numSpans int, err error) {
	numSent, numFailedToSend := toNumItems(numSpans, err)
	or.recordMetrics(noCancellationContext{Context: ctx}, component.DataTypeTraces, numSent, numFailedToSend)
	endSpan(ctx, err, numSent, numFailedToSend, obsmetrics.SentSpansKey, obsmetrics.FailedToSendSpansKey)
}

// StartMetricsOp is called at the start of an Export operation.
// The returned context should be used in other calls to the Exporter functions
// dealing with the same export operation.
func (or *ObsReport) StartMetricsOp(ctx context.Context) context.Context {
	return or.startOp(ctx, obsmetrics.ExportMetricsOperationSuffix)
}

// EndMetricsOp completes the export operation that was started with
// StartMetricsOp.
func (or *ObsReport) EndMetricsOp(ctx context.Context, numMetricPoints int, err error) {
	numSent, numFailedToSend := toNumItems(numMetricPoints, err)
	or.recordMetrics(noCancellationContext{Context: ctx}, component.DataTypeMetrics, numSent, numFailedToSend)
	endSpan(ctx, err, numSent, numFailedToSend, obsmetrics.SentMetricPointsKey, obsmetrics.FailedToSendMetricPointsKey)
}

// StartLogsOp is called at the start of an Export operation.
// The returned context should be used in other calls to the Exporter functions
// dealing with the same export operation.
func (or *ObsReport) StartLogsOp(ctx context.Context) context.Context {
	return or.startOp(ctx, obsmetrics.ExportLogsOperationSuffix)
}

// EndLogsOp completes the export operation that was started with StartLogsOp.
func (or *ObsReport) EndLogsOp(ctx context.Context, numLogRecords int, err error) {
	numSent, numFailedToSend := toNumItems(numLogRecords, err)
	or.recordMetrics(noCancellationContext{Context: ctx}, component.DataTypeLogs, numSent, numFailedToSend)
	endSpan(ctx, err, numSent, numFailedToSend, obsmetrics.SentLogRecordsKey, obsmetrics.FailedToSendLogRecordsKey)
}

// startOp creates the span used to trace the operation. Returning
// the updated context and the created span.
func (or *ObsReport) startOp(ctx context.Context, operationSuffix string) context.Context {
	spanName := or.spanNamePrefix + operationSuffix
	ctx, _ = or.tracer.Start(ctx, spanName)
	return ctx
}

func (or *ObsReport) recordMetrics(ctx context.Context, dataType component.DataType, sent, failed int64) {
	if or.level == configtelemetry.LevelNone {
		return
	}
	var sentMeasure, failedMeasure metric.Int64Counter
	switch dataType {
	case component.DataTypeTraces:
		sentMeasure = or.sentSpans
		failedMeasure = or.failedToSendSpans
	case component.DataTypeMetrics:
		sentMeasure = or.sentMetricPoints
		failedMeasure = or.failedToSendMetricPoints
	case component.DataTypeLogs:
		sentMeasure = or.sentLogRecords
		failedMeasure = or.failedToSendLogRecords
	}

	sentMeasure.Add(ctx, sent, metric.WithAttributes(or.otelAttrs...))
	failedMeasure.Add(ctx, failed, metric.WithAttributes(or.otelAttrs...))
}

func endSpan(ctx context.Context, err error, numSent, numFailedToSend int64, sentItemsKey, failedToSendItemsKey string) {
	span := trace.SpanFromContext(ctx)
	// End the span according to errors.
	if span.IsRecording() {
		span.SetAttributes(
			attribute.Int64(sentItemsKey, numSent),
			attribute.Int64(failedToSendItemsKey, numFailedToSend),
		)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		}
	}
	span.End()
}

func toNumItems(numExportedItems int, err error) (int64, int64) {
	if err != nil {
		return 0, int64(numExportedItems)
	}
	return int64(numExportedItems), 0
}

func (or *ObsReport) recordEnqueueFailure(ctx context.Context, dataType component.DataType, failed int64) {
	var enqueueFailedMeasure metric.Int64Counter
	switch dataType {
	case component.DataTypeTraces:
		enqueueFailedMeasure = or.failedToEnqueueSpans
	case component.DataTypeMetrics:
		enqueueFailedMeasure = or.failedToEnqueueMetricPoints
	case component.DataTypeLogs:
		enqueueFailedMeasure = or.failedToEnqueueLogRecords
	}

	enqueueFailedMeasure.Add(ctx, failed, metric.WithAttributes(or.otelAttrs...))
}
