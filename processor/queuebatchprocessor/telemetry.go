// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatchprocessor // import "go.opentelemetry.io/collector/processor/queuebatchprocessor"

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	queuebatchtelemetry "go.opentelemetry.io/collector/internal/telemetry/queuebatch"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor/queuebatchprocessor/internal/metadata"
)

const (
	processorKey = "processor"
	dataTypeKey  = "data_type"
	sizerKey     = "sizer"
)

func recordSize(inst, bytesInst metric.Int64Histogram, attrs metric.MeasurementOption) func(context.Context, int64, func() int64) {
	return func(ctx context.Context, items int64, bytesSize func() int64) {
		inst.Record(ctx, items, attrs)
		if bytesInst.Enabled(ctx) {
			bytesInst.Record(ctx, bytesSize(), attrs)
		}
	}
}

func newObsMetrics(
	set component.TelemetrySettings,
	id component.ID,
	signal pipeline.Signal,
	sizer exporterhelper.RequestSizerType,
) (queuebatchtelemetry.ObsMetrics, error) {
	tb, err := metadata.NewTelemetryBuilder(set)
	if err != nil {
		return queuebatchtelemetry.ObsMetrics{}, err
	}

	attrs := metric.WithAttributeSet(attribute.NewSet(
		attribute.String(processorKey, id.String()),
		attribute.String(dataTypeKey, signal.String()),
	))
	queueAttrs := metric.WithAttributeSet(attribute.NewSet(
		attribute.String(processorKey, id.String()),
		attribute.String(dataTypeKey, signal.String()),
		attribute.String(sizerKey, sizer.String()),
	))
	shutdown := sync.OnceFunc(tb.Shutdown)

	return queuebatchtelemetry.ObsMetrics{
		EnqueueFailureFunc: func(ctx context.Context, items int64) {
			tb.ProcessorQueuebatchEnqueueFailedItems.Add(ctx, items, attrs)
		},
		EnqueueSizeFunc: recordSize(tb.ProcessorQueuebatchEnqueueSize, tb.ProcessorQueuebatchEnqueueSizeBytes, attrs),
		RegisterQueueFunc: func(size, capacity func() int64) error {
			if err := tb.RegisterProcessorQueuebatchQueueSizeCallback(func(_ context.Context, o metric.Int64Observer) error {
				o.Observe(size(), queueAttrs)
				return nil
			}); err != nil {
				shutdown()
				return err
			}
			if err := tb.RegisterProcessorQueuebatchQueueCapacityCallback(func(_ context.Context, o metric.Int64Observer) error {
				o.Observe(capacity(), queueAttrs)
				return nil
			}); err != nil {
				shutdown()
				return err
			}
			return nil
		},
		BatchSendSizeFunc: recordSize(tb.ProcessorQueuebatchBatchSendSize, tb.ProcessorQueuebatchBatchSendSizeBytes, attrs),
		InFlightFunc: func(ctx context.Context, delta int64) {
			tb.ProcessorQueuebatchInFlightRequests.Add(ctx, delta, attrs)
		},
		SentFunc: func(ctx context.Context, items int64) {
			tb.ProcessorQueuebatchSentItems.Add(ctx, items, attrs)
		},
		SendFailureFunc: func(ctx context.Context, items int64, options ...metric.AddOption) {
			tb.ProcessorQueuebatchSendFailedItems.Add(ctx, items, append([]metric.AddOption{attrs}, options...)...)
		},
		ShutdownFunc: shutdown,
	}, nil
}
