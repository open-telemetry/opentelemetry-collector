// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper // import "go.opentelemetry.io/collector/processor/processorhelper"

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper/internal/metadata"
)

// BuildCustomMetricName is used to be build a metric name following
// the standards used in the Collector. The configType should be the same
// value used to identify the type on the config.
func BuildCustomMetricName(configType, metric string) string {
	componentPrefix := obsmetrics.ProcessorMetricPrefix
	if !strings.HasSuffix(componentPrefix, obsmetrics.MetricNameSep) {
		componentPrefix += obsmetrics.MetricNameSep
	}
	if configType == "" {
		return componentPrefix
	}
	return componentPrefix + configType + obsmetrics.MetricNameSep + metric
}

// ObsReport is a helper to add observability to a processor.
type ObsReport struct {
	otelAttrs        []attribute.KeyValue
	telemetryBuilder *metadata.TelemetryBuilder
}

// ObsReportSettings are settings for creating an ObsReport.
type ObsReportSettings struct {
	ProcessorID             component.ID
	ProcessorCreateSettings processor.Settings
}

// NewObsReport creates a new Processor.
func NewObsReport(cfg ObsReportSettings) (*ObsReport, error) {
	return newObsReport(cfg)
}

func newObsReport(cfg ObsReportSettings) (*ObsReport, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(cfg.ProcessorCreateSettings.TelemetrySettings, metadata.WithLevel(cfg.ProcessorCreateSettings.MetricsLevel))
	if err != nil {
		return nil, err
	}
	return &ObsReport{
		otelAttrs: []attribute.KeyValue{
			attribute.String(obsmetrics.ProcessorKey, cfg.ProcessorID.String()),
		},
		telemetryBuilder: telemetryBuilder,
	}, nil
}

func (or *ObsReport) recordData(ctx context.Context, dataType component.DataType, accepted, refused, dropped int64) {
	var acceptedCount, refusedCount, droppedCount metric.Int64Counter
	switch dataType {
	case component.DataTypeTraces:
		acceptedCount = or.telemetryBuilder.ProcessorAcceptedSpans
		refusedCount = or.telemetryBuilder.ProcessorRefusedSpans
		droppedCount = or.telemetryBuilder.ProcessorDroppedSpans
	case component.DataTypeMetrics:
		acceptedCount = or.telemetryBuilder.ProcessorAcceptedMetricPoints
		refusedCount = or.telemetryBuilder.ProcessorRefusedMetricPoints
		droppedCount = or.telemetryBuilder.ProcessorDroppedMetricPoints
	case component.DataTypeLogs:
		acceptedCount = or.telemetryBuilder.ProcessorAcceptedLogRecords
		refusedCount = or.telemetryBuilder.ProcessorRefusedLogRecords
		droppedCount = or.telemetryBuilder.ProcessorDroppedLogRecords
	}

	acceptedCount.Add(ctx, accepted, metric.WithAttributes(or.otelAttrs...))
	refusedCount.Add(ctx, refused, metric.WithAttributes(or.otelAttrs...))
	droppedCount.Add(ctx, dropped, metric.WithAttributes(or.otelAttrs...))
}

// TracesAccepted reports that the trace data was accepted.
func (or *ObsReport) TracesAccepted(ctx context.Context, numSpans int) {
	or.recordData(ctx, component.DataTypeTraces, int64(numSpans), int64(0), int64(0))
}

// TracesRefused reports that the trace data was refused.
func (or *ObsReport) TracesRefused(ctx context.Context, numSpans int) {
	or.recordData(ctx, component.DataTypeTraces, int64(0), int64(numSpans), int64(0))
}

// TracesDropped reports that the trace data was dropped.
func (or *ObsReport) TracesDropped(ctx context.Context, numSpans int) {
	or.recordData(ctx, component.DataTypeTraces, int64(0), int64(0), int64(numSpans))
}

// MetricsAccepted reports that the metrics were accepted.
func (or *ObsReport) MetricsAccepted(ctx context.Context, numPoints int) {
	or.recordData(ctx, component.DataTypeMetrics, int64(numPoints), int64(0), int64(0))
}

// MetricsRefused reports that the metrics were refused.
func (or *ObsReport) MetricsRefused(ctx context.Context, numPoints int) {
	or.recordData(ctx, component.DataTypeMetrics, int64(0), int64(numPoints), int64(0))
}

// MetricsDropped reports that the metrics were dropped.
func (or *ObsReport) MetricsDropped(ctx context.Context, numPoints int) {
	or.recordData(ctx, component.DataTypeMetrics, int64(0), int64(0), int64(numPoints))
}

// LogsAccepted reports that the logs were accepted.
func (or *ObsReport) LogsAccepted(ctx context.Context, numRecords int) {
	or.recordData(ctx, component.DataTypeLogs, int64(numRecords), int64(0), int64(0))
}

// LogsRefused reports that the logs were refused.
func (or *ObsReport) LogsRefused(ctx context.Context, numRecords int) {
	or.recordData(ctx, component.DataTypeLogs, int64(0), int64(numRecords), int64(0))
}

// LogsDropped reports that the logs were dropped.
func (or *ObsReport) LogsDropped(ctx context.Context, numRecords int) {
	or.recordData(ctx, component.DataTypeLogs, int64(0), int64(0), int64(numRecords))
}
