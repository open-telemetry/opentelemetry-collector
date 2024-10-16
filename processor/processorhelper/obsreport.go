// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper // import "go.opentelemetry.io/collector/processor/processorhelper"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/internal"
	"go.opentelemetry.io/collector/processor/processorhelper/internal/metadata"
)

const signalKey = "otel.signal"

type obsReport struct {
	otelAttrs        metric.MeasurementOption
	telemetryBuilder *metadata.TelemetryBuilder
}

func newObsReport(set processor.Settings, signal pipeline.Signal) (*obsReport, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	return &obsReport{
		otelAttrs: metric.WithAttributeSet(attribute.NewSet(
			attribute.String(internal.ProcessorKey, set.ID.String()),
			attribute.String(signalKey, signal.String()),
		)),
		telemetryBuilder: telemetryBuilder,
	}, nil
}

func (or *obsReport) recordInOut(ctx context.Context, incoming, outgoing int) {
	or.telemetryBuilder.ProcessorIncomingItems.Add(ctx, int64(incoming), or.otelAttrs)
	or.telemetryBuilder.ProcessorOutgoingItems.Add(ctx, int64(outgoing), or.otelAttrs)
}
