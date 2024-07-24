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
	otelAttrs  []attribute.KeyValue
	telBuilder *metadata.TelemetryBuilder
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
	telBuilder, err := metadata.NewTelemetryBuilder(cfg.ProcessorCreateSettings.TelemetrySettings, metadata.WithLevel(cfg.ProcessorCreateSettings.MetricsLevel))
	if err != nil {
		return nil, err
	}
	return &ObsReport{
		otelAttrs: []attribute.KeyValue{
			attribute.String(obsmetrics.ProcessorKey, cfg.ProcessorID.String()),
		},
		telBuilder: telBuilder,
	}, nil
}

// TracesAccepted reports that the trace data was accepted.
func (or *ObsReport) TracesAccepted(ctx context.Context, numSpans int) {
	or.telBuilder.ProcessorAcceptedSpans.Add(ctx, int64(numSpans), metric.WithAttributes(or.otelAttrs...))
}

// TracesRefused reports that the trace data was refused.
func (or *ObsReport) TracesRefused(ctx context.Context, numSpans int) {
	or.telBuilder.ProcessorRefusedSpans.Add(ctx, int64(numSpans), metric.WithAttributes(or.otelAttrs...))
}

// TracesDropped reports that the trace data was dropped.
func (or *ObsReport) TracesDropped(ctx context.Context, numSpans int) {
	or.telBuilder.ProcessorDroppedSpans.Add(ctx, int64(numSpans), metric.WithAttributes(or.otelAttrs...))
}

// TracesInserted reports that the trace data was inserted.
func (or *ObsReport) TracesInserted(ctx context.Context, numSpans int) {
	or.telBuilder.ProcessorInsertedSpans.Add(ctx, int64(numSpans), metric.WithAttributes(or.otelAttrs...))
}

// MetricsAccepted reports that the metrics were accepted.
func (or *ObsReport) MetricsAccepted(ctx context.Context, numPoints int) {
	or.telBuilder.ProcessorAcceptedMetricPoints.Add(ctx, int64(numPoints), metric.WithAttributes(or.otelAttrs...))
}

// MetricsRefused reports that the metrics were refused.
func (or *ObsReport) MetricsRefused(ctx context.Context, numPoints int) {
	or.telBuilder.ProcessorRefusedMetricPoints.Add(ctx, int64(numPoints), metric.WithAttributes(or.otelAttrs...))
}

// MetricsDropped reports that the metrics were dropped.
func (or *ObsReport) MetricsDropped(ctx context.Context, numPoints int) {
	or.telBuilder.ProcessorDroppedMetricPoints.Add(ctx, int64(numPoints), metric.WithAttributes(or.otelAttrs...))
}

// MetricsInserted reports that the metrics were inserted.
func (or *ObsReport) MetricsInserted(ctx context.Context, numPoints int) {
	or.telBuilder.ProcessorInsertedMetricPoints.Add(ctx, int64(numPoints), metric.WithAttributes(or.otelAttrs...))
}

// LogsAccepted reports that the logs were accepted.
func (or *ObsReport) LogsAccepted(ctx context.Context, numRecords int) {
	or.telBuilder.ProcessorAcceptedLogRecords.Add(ctx, int64(numRecords), metric.WithAttributes(or.otelAttrs...))
}

// LogsRefused reports that the logs were refused.
func (or *ObsReport) LogsRefused(ctx context.Context, numRecords int) {
	or.telBuilder.ProcessorRefusedLogRecords.Add(ctx, int64(numRecords), metric.WithAttributes(or.otelAttrs...))
}

// LogsDropped reports that the logs were dropped.
func (or *ObsReport) LogsDropped(ctx context.Context, numRecords int) {
	or.telBuilder.ProcessorDroppedLogRecords.Add(ctx, int64(numRecords), metric.WithAttributes(or.otelAttrs...))
}

// LogsInserted reports that the logs were inserted.
func (or *ObsReport) LogsInserted(ctx context.Context, numRecords int) {
	or.telBuilder.ProcessorInsertedLogRecords.Add(ctx, int64(numRecords), metric.WithAttributes(or.otelAttrs...))
}
