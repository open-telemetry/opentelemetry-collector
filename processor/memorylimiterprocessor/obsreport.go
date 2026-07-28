// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiterprocessor // import "go.opentelemetry.io/collector/processor/memorylimiterprocessor"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/internal"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor/internal/metadata"
)

type obsReport struct {
	otelAttrs        metric.MeasurementOption
	telemetryBuilder *metadata.TelemetryBuilder
}

func newObsReport(set processor.Settings) (*obsReport, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	return &obsReport{
		otelAttrs:        metric.WithAttributeSet(attribute.NewSet(attribute.String(internal.ProcessorKey, set.ID.String()))),
		telemetryBuilder: telemetryBuilder,
	}, nil
}

// accepted reports that the num data was accepted.
func (or *obsReport) accepted(ctx context.Context, num int, signal pipeline.Signal) {
	switch signal {
	case pipeline.SignalTraces:
		or.telemetryBuilder.ProcessorMemoryLimiterAcceptedSpans.Add(ctx, int64(num), or.otelAttrs)
	case pipeline.SignalMetrics:
		or.telemetryBuilder.ProcessorMemoryLimiterAcceptedMetricPoints.Add(ctx, int64(num), or.otelAttrs)
	case pipeline.SignalLogs:
		or.telemetryBuilder.ProcessorMemoryLimiterAcceptedLogRecords.Add(ctx, int64(num), or.otelAttrs)
	}
}

// refused reports that the num data was refused.
func (or *obsReport) refused(ctx context.Context, num int, signal pipeline.Signal) {
	switch signal {
	case pipeline.SignalTraces:
		or.telemetryBuilder.ProcessorMemoryLimiterRefusedSpans.Add(ctx, int64(num), or.otelAttrs)
	case pipeline.SignalMetrics:
		or.telemetryBuilder.ProcessorMemoryLimiterRefusedMetricPoints.Add(ctx, int64(num), or.otelAttrs)
	case pipeline.SignalLogs:
		or.telemetryBuilder.ProcessorMemoryLimiterRefusedLogRecords.Add(ctx, int64(num), or.otelAttrs)
	}
}
