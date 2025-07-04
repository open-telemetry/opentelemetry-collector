// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper // import "go.opentelemetry.io/collector/processor/processorhelper"

import (
	"context"

	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/internal"
	"go.opentelemetry.io/collector/processor/processorhelper/internal/metadata"
	"go.opentelemetry.io/otel/attribute"
)

const signalKey = "otel.signal"

type obsReport struct {
	telemetryBuilder *metadata.TelemetryBuilder
}

func newObsReport(set processor.Settings, signal pipeline.Signal) (*obsReport, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings, metadata.WithAttributeSet(attribute.NewSet(
		attribute.String(internal.ProcessorKey, set.ID.String()),
		attribute.String(signalKey, signal.String()),
	)))
	if err != nil {
		return nil, err
	}
	return &obsReport{
		telemetryBuilder: telemetryBuilder,
	}, nil
}

func (or *obsReport) recordInOut(ctx context.Context, incoming, outgoing int) {
	or.telemetryBuilder.RecordProcessorIncomingItemsDataPoint(ctx, int64(incoming))
	or.telemetryBuilder.RecordProcessorOutgoingItemsDataPoint(ctx, int64(outgoing))
}
