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

const (
	// spanNameSep is duplicate between receiver and exporter.
	spanNameSep = "/"

	// ExporterKey used to identify exporters in metrics and traces.
	ExporterKey = "exporter"

	// DataTypeKey used to identify the data type in the queue size metric.
	DataTypeKey = "data_type"

	// ItemsSent used to track number of items sent by exporters.
	ItemsSent = "items.sent"
	// ItemsFailed used to track number of items that failed to be sent by exporters.
	ItemsFailed = "items.failed"
)

// ObsReport is a helper to add observability to an exporter.
type ObsReport struct {
	spanName string
	tracer   trace.Tracer

	spanAttrs       trace.SpanStartEventOption
	metricAttr      metric.MeasurementOption
	itemsSentInst   metric.Int64Counter
	itemsFailedInst metric.Int64Counter
}

// ObsReportSettings are settings for creating an ObsReport.
type ObsReportSettings struct {
	ExporterSettings exporter.Settings
	Signal           pipeline.Signal
}

func NewObsReport(set ObsReportSettings) (*ObsReport, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.ExporterSettings.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	idStr := set.ExporterSettings.ID.String()
	expAttr := attribute.String(ExporterKey, idStr)

	or := &ObsReport{
		spanName:   ExporterKey + spanNameSep + idStr + spanNameSep + set.Signal.String(),
		tracer:     metadata.Tracer(set.ExporterSettings.TelemetrySettings),
		spanAttrs:  trace.WithAttributes(expAttr, attribute.String(DataTypeKey, set.Signal.String())),
		metricAttr: metric.WithAttributeSet(attribute.NewSet(expAttr)),
	}

	switch set.Signal {
	case pipeline.SignalTraces:
		or.itemsSentInst = telemetryBuilder.ExporterSentSpans
		or.itemsFailedInst = telemetryBuilder.ExporterSendFailedSpans

	case pipeline.SignalMetrics:
		or.itemsSentInst = telemetryBuilder.ExporterSentMetricPoints
		or.itemsFailedInst = telemetryBuilder.ExporterSendFailedMetricPoints

	case pipeline.SignalLogs:
		or.itemsSentInst = telemetryBuilder.ExporterSentLogRecords
		or.itemsFailedInst = telemetryBuilder.ExporterSendFailedLogRecords
	}

	return or, nil
}

// StartOp creates the span used to trace the operation. Returning
// the updated context and the created span.
func (or *ObsReport) StartOp(ctx context.Context) context.Context {
	ctx, _ = or.tracer.Start(ctx, or.spanName, or.spanAttrs)
	return ctx
}

// EndOp completes the export operation that was started with StartOp.
func (or *ObsReport) EndOp(ctx context.Context, numLogRecords int, err error) {
	numSent, numFailedToSend := toNumItems(numLogRecords, err)

	// No metrics recorded for profiles.
	if or.itemsSentInst != nil {
		or.itemsSentInst.Add(ctx, numSent, or.metricAttr)
	}
	// No metrics recorded for profiles.
	if or.itemsFailedInst != nil {
		or.itemsFailedInst.Add(ctx, numFailedToSend, or.metricAttr)
	}

	span := trace.SpanFromContext(ctx)
	defer span.End()
	// End the span according to errors.
	if span.IsRecording() {
		span.SetAttributes(
			attribute.Int64(ItemsSent, numSent),
			attribute.Int64(ItemsFailed, numFailedToSend),
		)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		}
	}
}

func toNumItems(numExportedItems int, err error) (int64, int64) {
	if err != nil {
		return 0, int64(numExportedItems)
	}
	return int64(numExportedItems), 0
}
