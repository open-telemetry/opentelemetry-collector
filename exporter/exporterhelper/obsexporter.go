// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

// obsReport is a helper to add observability to an exporter.
type obsReport struct {
	spanNamePrefix string
	tracer         trace.Tracer
	dataType       component.DataType

	otelAttrs        []attribute.KeyValue
	telemetryBuilder *metadata.TelemetryBuilder
}

// obsReportSettings are settings for creating an obsReport.
type obsReportSettings struct {
	exporterID             component.ID
	exporterCreateSettings exporter.Settings
	dataType               component.DataType
}

func newExporter(cfg obsReportSettings) (*obsReport, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(cfg.exporterCreateSettings.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	return &obsReport{
		spanNamePrefix: obsmetrics.ExporterPrefix + cfg.exporterID.String(),
		tracer:         cfg.exporterCreateSettings.TracerProvider.Tracer(cfg.exporterID.String()),
		dataType:       cfg.dataType,
		otelAttrs: []attribute.KeyValue{
			attribute.String(obsmetrics.ExporterKey, cfg.exporterID.String()),
		},
		telemetryBuilder: telemetryBuilder,
	}, nil
}

// startTracesOp is called at the start of an Export operation.
// The returned context should be used in other calls to the Exporter functions
// dealing with the same export operation.
func (or *obsReport) startTracesOp(ctx context.Context) context.Context {
	return or.startOp(ctx, obsmetrics.ExportTraceDataOperationSuffix)
}

// endTracesOp completes the export operation that was started with startTracesOp.
func (or *obsReport) endTracesOp(ctx context.Context, numSpans int, err error) {
	numSent, numFailedToSend := toNumItems(numSpans, err)
	or.recordMetrics(context.WithoutCancel(ctx), component.DataTypeTraces, numSent, numFailedToSend)
	endSpan(ctx, err, numSent, numFailedToSend, obsmetrics.SentSpansKey, obsmetrics.FailedToSendSpansKey)
}

// startMetricsOp is called at the start of an Export operation.
// The returned context should be used in other calls to the Exporter functions
// dealing with the same export operation.
func (or *obsReport) startMetricsOp(ctx context.Context) context.Context {
	return or.startOp(ctx, obsmetrics.ExportMetricsOperationSuffix)
}

// endMetricsOp completes the export operation that was started with
// startMetricsOp.
//
// If needed, report your use case in https://github.com/open-telemetry/opentelemetry-collector/issues/10592.
func (or *obsReport) endMetricsOp(ctx context.Context, numMetricPoints int, err error) {
	numSent, numFailedToSend := toNumItems(numMetricPoints, err)
	or.recordMetrics(context.WithoutCancel(ctx), component.DataTypeMetrics, numSent, numFailedToSend)
	endSpan(ctx, err, numSent, numFailedToSend, obsmetrics.SentMetricPointsKey, obsmetrics.FailedToSendMetricPointsKey)
}

// startLogsOp is called at the start of an Export operation.
// The returned context should be used in other calls to the Exporter functions
// dealing with the same export operation.
func (or *obsReport) startLogsOp(ctx context.Context) context.Context {
	return or.startOp(ctx, obsmetrics.ExportLogsOperationSuffix)
}

// endLogsOp completes the export operation that was started with startLogsOp.
func (or *obsReport) endLogsOp(ctx context.Context, numLogRecords int, err error) {
	numSent, numFailedToSend := toNumItems(numLogRecords, err)
	or.recordMetrics(context.WithoutCancel(ctx), component.DataTypeLogs, numSent, numFailedToSend)
	endSpan(ctx, err, numSent, numFailedToSend, obsmetrics.SentLogRecordsKey, obsmetrics.FailedToSendLogRecordsKey)
}

// startOp creates the span used to trace the operation. Returning
// the updated context and the created span.
func (or *obsReport) startOp(ctx context.Context, operationSuffix string) context.Context {
	spanName := or.spanNamePrefix + operationSuffix
	ctx, _ = or.tracer.Start(ctx, spanName)
	return ctx
}

func (or *obsReport) recordMetrics(ctx context.Context, dataType component.DataType, sent, failed int64) {
	var sentMeasure, failedMeasure metric.Int64Counter
	switch dataType {
	case component.DataTypeTraces:
		sentMeasure = or.telemetryBuilder.ExporterSentSpans
		failedMeasure = or.telemetryBuilder.ExporterSendFailedSpans
	case component.DataTypeMetrics:
		sentMeasure = or.telemetryBuilder.ExporterSentMetricPoints
		failedMeasure = or.telemetryBuilder.ExporterSendFailedMetricPoints
	case component.DataTypeLogs:
		sentMeasure = or.telemetryBuilder.ExporterSentLogRecords
		failedMeasure = or.telemetryBuilder.ExporterSendFailedLogRecords
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

func (or *obsReport) recordEnqueueFailure(ctx context.Context, dataType component.DataType, failed int64) {
	var enqueueFailedMeasure metric.Int64Counter
	switch dataType {
	case component.DataTypeTraces:
		enqueueFailedMeasure = or.telemetryBuilder.ExporterEnqueueFailedSpans
	case component.DataTypeMetrics:
		enqueueFailedMeasure = or.telemetryBuilder.ExporterEnqueueFailedMetricPoints
	case component.DataTypeLogs:
		enqueueFailedMeasure = or.telemetryBuilder.ExporterEnqueueFailedLogRecords
	}

	enqueueFailedMeasure.Add(ctx, failed, metric.WithAttributes(or.otelAttrs...))
}
