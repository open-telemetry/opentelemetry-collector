// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper // import "go.opentelemetry.io/collector/processor/processorhelper"

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/component"
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
