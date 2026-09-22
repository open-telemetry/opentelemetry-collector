// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatchprocessor // import "go.opentelemetry.io/collector/processor/queuebatchprocessor"

import (
	"context"
	"fmt"
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
		ShouldRecordFunc: func(ctx context.Context, m queuebatchtelemetry.Metric) bool {
			switch m {
			case queuebatchtelemetry.MetricEnqueueSizeBytes:
				return tb.ProcessorQueuebatchEnqueueSizeBytes.Enabled(ctx)
			case queuebatchtelemetry.MetricBatchSendSizeBytes:
				return tb.ProcessorQueuebatchBatchSendSizeBytes.Enabled(ctx)
			default:
				panic(fmt.Sprintf("unsupported optional queuebatch metric %q", m))
			}
		},
		RecordIntFunc: func(ctx context.Context, m queuebatchtelemetry.Metric, value int64, options ...metric.AddOption) {
			switch m {
			case queuebatchtelemetry.MetricEnqueueFailure:
				tb.ProcessorQueuebatchEnqueueFailedItems.Add(ctx, value, attrs)
			case queuebatchtelemetry.MetricEnqueueSize:
				tb.ProcessorQueuebatchEnqueueSize.Record(ctx, value, attrs)
			case queuebatchtelemetry.MetricEnqueueSizeBytes:
				tb.ProcessorQueuebatchEnqueueSizeBytes.Record(ctx, value, attrs)
			case queuebatchtelemetry.MetricBatchSendSize:
				tb.ProcessorQueuebatchBatchSendSize.Record(ctx, value, attrs)
			case queuebatchtelemetry.MetricBatchSendSizeBytes:
				tb.ProcessorQueuebatchBatchSendSizeBytes.Record(ctx, value, attrs)
			case queuebatchtelemetry.MetricInFlight:
				tb.ProcessorQueuebatchInFlightRequests.Add(ctx, value, attrs)
			case queuebatchtelemetry.MetricSent:
				tb.ProcessorQueuebatchSentItems.Add(ctx, value, attrs)
			case queuebatchtelemetry.MetricSendFailure:
				tb.ProcessorQueuebatchSendFailedItems.Add(ctx, value, append([]metric.AddOption{attrs}, options...)...)
			default:
				panic(fmt.Sprintf("unsupported queuebatch metric %q", m))
			}
		},
		RegisterIntFunc: func(m queuebatchtelemetry.Metric, value func() int64) error {
			var err error
			switch m {
			case queuebatchtelemetry.MetricQueueSize:
				err = tb.RegisterProcessorQueuebatchQueueSizeCallback(func(_ context.Context, o metric.Int64Observer) error {
					o.Observe(value(), queueAttrs)
					return nil
				})
			case queuebatchtelemetry.MetricQueueCapacity:
				err = tb.RegisterProcessorQueuebatchQueueCapacityCallback(func(_ context.Context, o metric.Int64Observer) error {
					o.Observe(value(), queueAttrs)
					return nil
				})
			default:
				return fmt.Errorf("unsupported observable queuebatch metric %q", m)
			}
			if err != nil {
				shutdown()
			}
			return err
		},
		ShutdownFunc: shutdown,
	}, nil
}
