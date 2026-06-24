// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatchprocessor // import "go.opentelemetry.io/collector/processor/queuebatchprocessor"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/processor/queuebatchprocessor/internal/metadata"
)

// processorKey identifies the processor in metrics, matching batchprocessor.
const processorKey = "processor"

// batchTelemetry records the batch metrics by observing batches as they are
// sent to the next consumer. The exporterhelper's own telemetry is disabled
// (see exporterSettings), so these are the only batch-related series emitted,
// and they use the same names as batchprocessor.
type batchTelemetry struct {
	telemetryBuilder *metadata.TelemetryBuilder
	attr             metric.MeasurementOption
}

func newBatchTelemetry(set component.TelemetrySettings, id component.ID) (*batchTelemetry, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set)
	if err != nil {
		return nil, err
	}
	return &batchTelemetry{
		telemetryBuilder: telemetryBuilder,
		attr:             metric.WithAttributeSet(attribute.NewSet(attribute.String(processorKey, id.String()))),
	}, nil
}

// recordBatch records the size of a batch that was sent downstream. items is the
// number of telemetry items in the batch; sizeBytes is a function that computes
// the encoded byte size, called only when the bytes histogram is enabled (the
// computation is expensive and only needed at detailed telemetry level).
func (bt *batchTelemetry) recordBatch(ctx context.Context, items int64, sizeBytes func() int64) {
	bt.telemetryBuilder.ProcessorBatchBatchSendSize.Record(ctx, items, bt.attr)

	inst := bt.telemetryBuilder.ProcessorBatchBatchSendSizeBytes
	if enabled, ok := inst.(interface{ Enabled(context.Context) bool }); !ok || enabled.Enabled(ctx) {
		inst.Record(ctx, sizeBytes(), bt.attr)
	}
}
