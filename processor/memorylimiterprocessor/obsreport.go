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

// refused reports that the num data that was refused.
func (or *obsReport) accepted(ctx context.Context, num int, signal pipeline.Signal) {
	switch signal {
	case pipeline.SignalTraces:
		or.telemetryBuilder.RecordProcessorAcceptedSpans(ctx, int64(num), or.otelAttrs)
	case pipeline.SignalMetrics:
		or.telemetryBuilder.RecordProcessorAcceptedMetricPoints(ctx, int64(num), or.otelAttrs)
	case pipeline.SignalLogs:
		or.telemetryBuilder.RecordProcessorAcceptedLogRecords(ctx, int64(num), or.otelAttrs)
	}
}

// refused reports that the num data that was refused.
func (or *obsReport) refused(ctx context.Context, num int, signal pipeline.Signal) {
	switch signal {
	case pipeline.SignalTraces:
		or.telemetryBuilder.RecordProcessorRefusedSpans(ctx, int64(num), or.otelAttrs)
	case pipeline.SignalMetrics:
		or.telemetryBuilder.RecordProcessorRefusedMetricPoints(ctx, int64(num), or.otelAttrs)
	case pipeline.SignalLogs:
		or.telemetryBuilder.RecordProcessorRefusedLogRecords(ctx, int64(num), or.otelAttrs)
	}
}
