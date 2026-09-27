// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package obsreport provides the processor telemetry instrumentation shared by
// processorhelper and xprocessorhelper.
package obsreport // import "go.opentelemetry.io/collector/processor/processorhelper/internal/obsreport"

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/internal"
	"go.opentelemetry.io/collector/processor/processorhelper/internal/metadata"
)

const signalKey = "otel.signal"

type ObsReport struct {
	otelAttrs        metric.MeasurementOption
	telemetryBuilder *metadata.TelemetryBuilder
}

func New(set processor.Settings, signal pipeline.Signal) (*ObsReport, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	return &ObsReport{
		otelAttrs: metric.WithAttributeSet(attribute.NewSet(
			attribute.String(internal.ProcessorKey, set.ID.String()),
			attribute.String(signalKey, signal.String()),
		)),
		telemetryBuilder: telemetryBuilder,
	}, nil
}

func (or *ObsReport) RecordInOut(ctx context.Context, incoming, outgoing int) {
	or.telemetryBuilder.ProcessorIncomingItems.Add(ctx, int64(incoming), or.otelAttrs)
	or.telemetryBuilder.ProcessorOutgoingItems.Add(ctx, int64(outgoing), or.otelAttrs)
}

func (or *ObsReport) RecordInternalDuration(ctx context.Context, startTime time.Time) {
	duration := time.Since(startTime)
	or.telemetryBuilder.ProcessorInternalDuration.Record(ctx, duration.Seconds(), or.otelAttrs)
}
