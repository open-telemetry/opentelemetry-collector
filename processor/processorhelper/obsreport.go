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

const signalKey = "otel.signal"

// Deprecated: [v0.111.0] no longer needed. To be removed in future.
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

// Deprecated: [v0.111.0] not used.
type ObsReport struct{}

// Deprecated: [v0.111.0] not used.
type ObsReportSettings struct {
	ProcessorID             component.ID
	ProcessorCreateSettings processor.Settings
}

// Deprecated: [v0.111.0] not used.
func NewObsReport(_ ObsReportSettings) (*ObsReport, error) {
	return &ObsReport{}, nil
}

type obsReport struct {
	otelAttrs        attribute.Set
	telemetryBuilder *metadata.TelemetryBuilder
}

func newObsReport(set processor.Settings, signal pipeline.Signal) (*obsReport, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	return &obsReport{
		otelAttrs: attribute.NewSet(
			attribute.String(internal.ProcessorKey, set.ID.String()),
			attribute.String(signalKey, signal.String()),
		),
		telemetryBuilder: telemetryBuilder,
	}, nil
}

func (or *obsReport) recordInOut(ctx context.Context, incoming, outgoing int) {
	or.telemetryBuilder.ProcessorIncomingItems.Add(ctx, int64(incoming), metric.WithAttributeSet(or.otelAttrs))
	or.telemetryBuilder.ProcessorOutgoingItems.Add(ctx, int64(outgoing), metric.WithAttributeSet(or.otelAttrs))
}
