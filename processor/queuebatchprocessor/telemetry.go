// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatchprocessor // import "go.opentelemetry.io/collector/processor/queuebatchprocessor"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor/queuebatchprocessor/internal/metadata"
)

const (
	// processorKey identifies the processor in metrics, matching processorhelper.
	processorKey = "processor"
	// signalKey identifies the pipeline signal in metrics, matching processorhelper.
	signalKey = "otel.signal"
)

// batchTelemetry records this processor's metrics. The exporterhelper's own
// telemetry is disabled (see exporterSettings), so these are the only series
// emitted. incoming/outgoing item counters mirror processorhelper; the
// queuebatch histograms are recorded only at the detailed telemetry level.
type batchTelemetry struct {
	telemetryBuilder *metadata.TelemetryBuilder
	attr             metric.MeasurementOption
}

func newBatchTelemetry(set component.TelemetrySettings, id component.ID, signal pipeline.Signal) (*batchTelemetry, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set)
	if err != nil {
		return nil, err
	}
	return &batchTelemetry{
		telemetryBuilder: telemetryBuilder,
		attr: metric.WithAttributeSet(attribute.NewSet(
			attribute.String(processorKey, id.String()),
			attribute.String(signalKey, signal.String()),
		)),
	}, nil
}

// recordIncoming counts items as they arrive at the processor.
func (bt *batchTelemetry) recordIncoming(ctx context.Context, items int64) {
	bt.telemetryBuilder.ProcessorIncomingItems.Add(ctx, items, bt.attr)
}

// recordOutgoing counts items and batch sizes as each batch is sent downstream.
// items is the number of items in the batch; sizeBytes computes the encoded
// byte size and is only called when the bytes histogram is enabled, since the
// computation is expensive and the histogram is collected only at detailed level.
func (bt *batchTelemetry) recordOutgoing(ctx context.Context, items int64, sizeBytes func() int64) {
	bt.telemetryBuilder.ProcessorOutgoingItems.Add(ctx, items, bt.attr)

	if instrumentEnabled(ctx, bt.telemetryBuilder.ProcessorQueuebatchItems) {
		bt.telemetryBuilder.ProcessorQueuebatchItems.Record(ctx, items, bt.attr)
	}
	if instrumentEnabled(ctx, bt.telemetryBuilder.ProcessorQueuebatchBytes) {
		bt.telemetryBuilder.ProcessorQueuebatchBytes.Record(ctx, sizeBytes(), bt.attr)
	}
}

// instrumentEnabled reports whether the instrument is currently collecting. It
// returns false when a view drops the instrument (e.g. below detailed level),
// letting callers skip both the record and any expensive value computation. An
// instrument that does not implement the check is treated as enabled.
func instrumentEnabled(ctx context.Context, inst any) bool {
	if e, ok := inst.(interface{ Enabled(context.Context) bool }); ok {
		return e.Enabled(ctx)
	}
	return true
}
