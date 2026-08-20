// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatchprocessor // import "go.opentelemetry.io/collector/processor/queuebatchprocessor"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor/queuebatchprocessor/internal/metadata"
)

const (
	processorKey = "processor"
	dataTypeKey  = "data_type"
)

// recordSize records the item count, and the byte size when bytesInst is enabled.
func recordSize(inst, bytesInst metric.Int64Histogram, attrs metric.MeasurementOption) func(context.Context, int64, func() int64) {
	return func(ctx context.Context, items int64, bytesSize func() int64) {
		inst.Record(ctx, items, attrs)
		if bytesInst.Enabled(ctx) {
			bytesInst.Record(ctx, bytesSize(), attrs)
		}
	}
}

// newObsMetrics reports exporterhelper's observation events through processor instruments.
func newObsMetrics(set component.TelemetrySettings, id component.ID, signal pipeline.Signal) (exporterhelper.ObsMetrics, error) {
	tb, err := metadata.NewTelemetryBuilder(set)
	if err != nil {
		return nil, err
	}

	attrs := metric.WithAttributeSet(attribute.NewSet(
		attribute.String(processorKey, id.String()),
		attribute.String(dataTypeKey, signal.String()),
	))

	return exporterhelper.NewObsMetrics(
		exporterhelper.NewQueueBatchMetrics(
			exporterhelper.NewQueueMetrics(
				func(ctx context.Context, items int64) {
					tb.ProcessorQueuebatchEnqueueFailedItems.Add(ctx, items, attrs)
				},
				recordSize(tb.ProcessorQueuebatchEnqueueSize, tb.ProcessorQueuebatchEnqueueSizeBytes, attrs),
				func(observeSize func() int64) error {
					return tb.RegisterProcessorQueuebatchQueueSizeCallback(func(_ context.Context, o metric.Int64Observer) error {
						o.Observe(observeSize(), attrs)
						return nil
					})
				},
				func(observeCapacity func() int64) error {
					return tb.RegisterProcessorQueuebatchQueueCapacityCallback(func(_ context.Context, o metric.Int64Observer) error {
						o.Observe(observeCapacity(), attrs)
						return nil
					})
				},
			),
			recordSize(tb.ProcessorQueuebatchBatchSendSize, tb.ProcessorQueuebatchBatchSendSizeBytes, attrs),
		),
		func(ctx context.Context, delta int64) {
			tb.ProcessorQueuebatchInFlightRequests.Add(ctx, delta, attrs)
		},
		func(ctx context.Context, items int64) {
			tb.ProcessorQueuebatchSentItems.Add(ctx, items, attrs)
		},
		func(ctx context.Context, items int64, options ...metric.AddOption) {
			tb.ProcessorQueuebatchSendFailedItems.Add(ctx, items, append([]metric.AddOption{attrs}, options...)...)
		},
		tb.Shutdown,
	), nil
}
