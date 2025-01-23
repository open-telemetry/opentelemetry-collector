// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/pipeline"
)

// ObsReport is a helper to add observability to an exporter.
type ObsReport struct {
	spanNamePrefix string
	tracer         trace.Tracer
	Signal         pipeline.Signal

	spanAttrs        trace.SpanStartEventOption
	metricsAttrs     metric.MeasurementOption
	TelemetryBuilder *metadata.TelemetryBuilder
}

// ObsReportSettings are settings for creating an ObsReport.
type ObsReportSettings struct {
	ExporterSettings exporter.Settings
	Signal           pipeline.Signal
}

func NewExporter(set ObsReportSettings) (*ObsReport, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.ExporterSettings.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	idStr := set.ExporterSettings.ID.String()
	expAttr := attribute.String(ExporterKey, idStr)

	return &ObsReport{
		spanNamePrefix:   ExporterPrefix + idStr,
		tracer:           metadata.Tracer(set.ExporterSettings.TelemetrySettings),
		Signal:           set.Signal,
		spanAttrs:        trace.WithAttributes(expAttr, attribute.String(DataTypeKey, set.Signal.String())),
		metricsAttrs:     metric.WithAttributeSet(attribute.NewSet(expAttr)),
		TelemetryBuilder: telemetryBuilder,
	}, nil
}

// StartTracesOp is called at the start of an Export operation.
// The returned context should be used in other calls to the Exporter functions
// dealing with the same export operation.
func (or *ObsReport) StartTracesOp(ctx context.Context) context.Context {
	return or.startOp(ctx, ExportTraceDataOperationSuffix)
}

// EndTracesOp completes the export operation that was started with startTracesOp.
func (or *ObsReport) EndTracesOp(ctx context.Context, numSpans int, err error) {
	numSent, numFailedToSend := toNumItems(numSpans, err)
	or.recordMetrics(context.WithoutCancel(ctx), pipeline.SignalTraces, numSent, numFailedToSend)
	endSpan(ctx, err, numSent, numFailedToSend, SentSpansKey, FailedToSendSpansKey)
}

// StartMetricsOp is called at the start of an Export operation.
// The returned context should be used in other calls to the Exporter functions
// dealing with the same export operation.
func (or *ObsReport) StartMetricsOp(ctx context.Context) context.Context {
	return or.startOp(ctx, ExportMetricsOperationSuffix)
}

// EndMetricsOp completes the export operation that was started with
// startMetricsOp.
//
// If needed, report your use case in https://github.com/open-telemetry/opentelemetry-collector/issues/10592.
func (or *ObsReport) EndMetricsOp(ctx context.Context, numMetricPoints int, err error) {
	numSent, numFailedToSend := toNumItems(numMetricPoints, err)
	or.recordMetrics(context.WithoutCancel(ctx), pipeline.SignalMetrics, numSent, numFailedToSend)
	endSpan(ctx, err, numSent, numFailedToSend, SentMetricPointsKey, FailedToSendMetricPointsKey)
}

// StartLogsOp is called at the start of an Export operation.
// The returned context should be used in other calls to the Exporter functions
// dealing with the same export operation.
func (or *ObsReport) StartLogsOp(ctx context.Context) context.Context {
	return or.startOp(ctx, ExportLogsOperationSuffix)
}

// EndLogsOp completes the export operation that was started with startLogsOp.
func (or *ObsReport) EndLogsOp(ctx context.Context, numLogRecords int, err error) {
	numSent, numFailedToSend := toNumItems(numLogRecords, err)
	or.recordMetrics(context.WithoutCancel(ctx), pipeline.SignalLogs, numSent, numFailedToSend)
	endSpan(ctx, err, numSent, numFailedToSend, SentLogRecordsKey, FailedToSendLogRecordsKey)
}

// StartProfilesOp is called at the start of an Export operation.
// The returned context should be used in other calls to the Exporter functions
// dealing with the same export operation.
func (or *ObsReport) StartProfilesOp(ctx context.Context) context.Context {
	return or.startOp(ctx, ExportTraceDataOperationSuffix)
}

// EndProfilesOp completes the export operation that was started with startProfilesOp.
func (or *ObsReport) EndProfilesOp(ctx context.Context, numSpans int, err error) {
	numSent, numFailedToSend := toNumItems(numSpans, err)
	endSpan(ctx, err, numSent, numFailedToSend, SentSamplesKey, FailedToSendSamplesKey)
}

// startOp creates the span used to trace the operation. Returning
// the updated context and the created span.
func (or *ObsReport) startOp(ctx context.Context, operationSuffix string) context.Context {
	spanName := or.spanNamePrefix + operationSuffix
	ctx, _ = or.tracer.Start(ctx, spanName, or.spanAttrs)
	return ctx
}

func (or *ObsReport) recordMetrics(ctx context.Context, signal pipeline.Signal, sent, failed int64) {
	var sentMeasure, failedMeasure metric.Int64Counter
	switch signal {
	case pipeline.SignalTraces:
		sentMeasure = or.TelemetryBuilder.ExporterSentSpans
		failedMeasure = or.TelemetryBuilder.ExporterSendFailedSpans
	case pipeline.SignalMetrics:
		sentMeasure = or.TelemetryBuilder.ExporterSentMetricPoints
		failedMeasure = or.TelemetryBuilder.ExporterSendFailedMetricPoints
	case pipeline.SignalLogs:
		sentMeasure = or.TelemetryBuilder.ExporterSentLogRecords
		failedMeasure = or.TelemetryBuilder.ExporterSendFailedLogRecords
	}

	sentMeasure.Add(ctx, sent, or.metricsAttrs)
	failedMeasure.Add(ctx, failed, or.metricsAttrs)
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

func (or *ObsReport) RecordEnqueueFailure(ctx context.Context, signal pipeline.Signal, failed int64) {
	var enqueueFailedMeasure metric.Int64Counter
	switch signal {
	case pipeline.SignalTraces:
		enqueueFailedMeasure = or.TelemetryBuilder.ExporterEnqueueFailedSpans
	case pipeline.SignalMetrics:
		enqueueFailedMeasure = or.TelemetryBuilder.ExporterEnqueueFailedMetricPoints
	case pipeline.SignalLogs:
		enqueueFailedMeasure = or.TelemetryBuilder.ExporterEnqueueFailedLogRecords
	}

	enqueueFailedMeasure.Add(ctx, failed, or.metricsAttrs)
}
