// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper // import "go.opentelemetry.io/collector/processor/processorhelper"

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/internal"
	"go.opentelemetry.io/collector/processor/processorhelper/internal/metadata"
)

// BuildCustomMetricName is used to be build a metric name following
// the standards used in the Collector. The configType should be the same
// value used to identify the type on the config.
func BuildCustomMetricName(configType, metric string) string {
	componentPrefix := internal.ProcessorMetricPrefix
	if !strings.HasSuffix(componentPrefix, internal.MetricNameSep) {
		componentPrefix += internal.MetricNameSep
	}
	if configType == "" {
		return componentPrefix
	}
	return componentPrefix + configType + internal.MetricNameSep + metric
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
	telemetryBuilder, err := metadata.NewTelemetryBuilder(cfg.ProcessorCreateSettings.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	return &ObsReport{
		otelAttrs: []attribute.KeyValue{
			attribute.String(internal.ProcessorKey, cfg.ProcessorID.String()),
		},
		telemetryBuilder: telemetryBuilder,
	}, nil
}

func (or *ObsReport) recordInOut(ctx context.Context, incoming, outgoing int) {
	or.telemetryBuilder.ProcessorIncomingItems.Add(ctx, int64(incoming), metric.WithAttributes(or.otelAttrs...))
	or.telemetryBuilder.ProcessorOutgoingItems.Add(ctx, int64(outgoing), metric.WithAttributes(or.otelAttrs...))
}

func (or *ObsReport) recordData(ctx context.Context, signal pipeline.Signal, accepted, refused, dropped int64) {
	var acceptedCount, refusedCount, droppedCount metric.Int64Counter
	switch signal {
	case pipeline.SignalTraces:
		acceptedCount = or.telemetryBuilder.ProcessorMemoryLimiterAcceptedSpans
		refusedCount = or.telemetryBuilder.ProcessorMemoryLimiterRefusedSpans
		droppedCount = or.telemetryBuilder.ProcessorMemoryLimiterDroppedSpans
	case pipeline.SignalMetrics:
		acceptedCount = or.telemetryBuilder.ProcessorMemoryLimiterAcceptedMetricPoints
		refusedCount = or.telemetryBuilder.ProcessorMemoryLimiterRefusedMetricPoints
		droppedCount = or.telemetryBuilder.ProcessorMemoryLimiterDroppedMetricPoints
	case pipeline.SignalLogs:
		acceptedCount = or.telemetryBuilder.ProcessorMemoryLimiterAcceptedLogRecords
		refusedCount = or.telemetryBuilder.ProcessorMemoryLimiterRefusedLogRecords
		droppedCount = or.telemetryBuilder.ProcessorMemoryLimiterDroppedLogRecords
	}

	acceptedCount.Add(ctx, accepted, metric.WithAttributes(or.otelAttrs...))
	refusedCount.Add(ctx, refused, metric.WithAttributes(or.otelAttrs...))
	droppedCount.Add(ctx, dropped, metric.WithAttributes(or.otelAttrs...))
}

// TracesAccepted reports that the trace data was accepted.
//
// Deprecated: [v0.110.0] Processor helper automatically calculates incoming/outgoing metrics only.
func (or *ObsReport) TracesAccepted(ctx context.Context, numSpans int) {
	or.recordData(ctx, pipeline.SignalTraces, int64(numSpans), int64(0), int64(0))
}

// TracesRefused reports that the trace data was refused.
//
// Deprecated: [v0.110.0] Processor helper automatically calculates incoming/outgoing metrics only.
func (or *ObsReport) TracesRefused(ctx context.Context, numSpans int) {
	or.recordData(ctx, pipeline.SignalTraces, int64(0), int64(numSpans), int64(0))
}

// TracesDropped reports that the trace data was dropped.
//
// Deprecated: [v0.110.0] Processor helper automatically calculates incoming/outgoing metrics only.
func (or *ObsReport) TracesDropped(ctx context.Context, numSpans int) {
	or.recordData(ctx, pipeline.SignalTraces, int64(0), int64(0), int64(numSpans))
}

// MetricsAccepted reports that the metrics were accepted.
//
// Deprecated: [v0.110.0] Processor helper automatically calculates incoming/outgoing metrics only.
func (or *ObsReport) MetricsAccepted(ctx context.Context, numPoints int) {
	or.recordData(ctx, pipeline.SignalMetrics, int64(numPoints), int64(0), int64(0))
}

// MetricsRefused reports that the metrics were refused.
//
// Deprecated: [v0.110.0] Processor helper automatically calculates incoming/outgoing metrics only.
func (or *ObsReport) MetricsRefused(ctx context.Context, numPoints int) {
	or.recordData(ctx, pipeline.SignalMetrics, int64(0), int64(numPoints), int64(0))
}

// MetricsDropped reports that the metrics were dropped.
//
// Deprecated: [v0.110.0] Processor helper automatically calculates incoming/outgoing metrics only.
func (or *ObsReport) MetricsDropped(ctx context.Context, numPoints int) {
	or.recordData(ctx, pipeline.SignalMetrics, int64(0), int64(0), int64(numPoints))
}

// LogsAccepted reports that the logs were accepted.
//
// Deprecated: [v0.110.0] Processor helper automatically calculates incoming/outgoing metrics only.
func (or *ObsReport) LogsAccepted(ctx context.Context, numRecords int) {
	or.recordData(ctx, pipeline.SignalLogs, int64(numRecords), int64(0), int64(0))
}

// LogsRefused reports that the logs were refused.
//
// Deprecated: [v0.110.0] Processor helper automatically calculates incoming/outgoing metrics only.
func (or *ObsReport) LogsRefused(ctx context.Context, numRecords int) {
	or.recordData(ctx, pipeline.SignalLogs, int64(0), int64(numRecords), int64(0))
}

// LogsDropped reports that the logs were dropped.
//
// Deprecated: [v0.110.0] Processor helper automatically calculates incoming/outgoing metrics only.
func (or *ObsReport) LogsDropped(ctx context.Context, numRecords int) {
	or.recordData(ctx, pipeline.SignalLogs, int64(0), int64(0), int64(numRecords))
}
