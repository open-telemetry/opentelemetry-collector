// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	queuebatchtelemetry "go.opentelemetry.io/collector/internal/telemetry/queuebatch"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

func NewExporterObsMetrics(
	telemetry component.TelemetrySettings,
	id component.ID,
	signal pipeline.Signal,
	extraAttrs []attribute.KeyValue,
) (queuebatchtelemetry.ObsMetrics, error) {
	tb, err := metadata.NewTelemetryBuilder(telemetry)
	if err != nil {
		return queuebatchtelemetry.ObsMetrics{}, err
	}

	exporterAttr := attribute.String(exporterKey, id.String())
	// These sets intentionally preserve existing metric cardinality. See
	// https://github.com/open-telemetry/opentelemetry-collector/issues/16049.
	enqueueAttr := metric.WithAttributeSet(attribute.NewSet(exporterAttr))
	senderAttr := metric.WithAttributeSet(attribute.NewSet(append(extraAttrs, exporterAttr)...))
	queueAttr := metric.WithAttributeSet(attribute.NewSet(
		exporterAttr,
		attribute.String(dataTypeKey, signal.String()),
	))
	shutdown := sync.OnceFunc(tb.Shutdown)

	var enqueueFailedInst, itemsSentInst, itemsFailedInst metric.Int64Counter
	switch signal {
	case pipeline.SignalTraces:
		enqueueFailedInst = tb.ExporterEnqueueFailedSpans
		itemsSentInst = tb.ExporterSentSpans
		itemsFailedInst = tb.ExporterSendFailedSpans
	case pipeline.SignalMetrics:
		enqueueFailedInst = tb.ExporterEnqueueFailedMetricPoints
		itemsSentInst = tb.ExporterSentMetricPoints
		itemsFailedInst = tb.ExporterSendFailedMetricPoints
	case pipeline.SignalLogs:
		enqueueFailedInst = tb.ExporterEnqueueFailedLogRecords
		itemsSentInst = tb.ExporterSentLogRecords
		itemsFailedInst = tb.ExporterSendFailedLogRecords
	case xpipeline.SignalProfiles:
		enqueueFailedInst = tb.ExporterEnqueueFailedProfileSamples
		itemsSentInst = tb.ExporterSentProfileSamples
		itemsFailedInst = tb.ExporterSendFailedProfileSamples
	}

	return queuebatchtelemetry.ObsMetrics{
		ShouldRecordFunc: func(ctx context.Context, m queuebatchtelemetry.Metric) bool {
			switch m {
			case queuebatchtelemetry.MetricEnqueueFailure:
				return enqueueFailedInst.Enabled(ctx)
			case queuebatchtelemetry.MetricEnqueueSize:
				return tb.ExporterEnqueueSize.Enabled(ctx)
			case queuebatchtelemetry.MetricEnqueueSizeBytes:
				return tb.ExporterEnqueueSizeBytes.Enabled(ctx)
			case queuebatchtelemetry.MetricBatchSendSize:
				return tb.ExporterQueueBatchSendSize.Enabled(ctx)
			case queuebatchtelemetry.MetricBatchSendSizeBytes:
				return tb.ExporterQueueBatchSendSizeBytes.Enabled(ctx)
			case queuebatchtelemetry.MetricInFlight:
				return tb.ExporterInFlightRequests.Enabled(ctx)
			case queuebatchtelemetry.MetricSent:
				return itemsSentInst.Enabled(ctx)
			case queuebatchtelemetry.MetricSendFailure:
				return itemsFailedInst.Enabled(ctx)
			}
			return false
		},
		RecordIntFunc: func(ctx context.Context, m queuebatchtelemetry.Metric, value int64, options ...metric.AddOption) {
			switch m {
			case queuebatchtelemetry.MetricEnqueueFailure:
				enqueueFailedInst.Add(ctx, value, enqueueAttr)
			case queuebatchtelemetry.MetricEnqueueSize:
				tb.ExporterEnqueueSize.Record(ctx, value, enqueueAttr)
			case queuebatchtelemetry.MetricEnqueueSizeBytes:
				tb.ExporterEnqueueSizeBytes.Record(ctx, value, enqueueAttr)
			case queuebatchtelemetry.MetricBatchSendSize:
				tb.ExporterQueueBatchSendSize.Record(ctx, value, senderAttr)
			case queuebatchtelemetry.MetricBatchSendSizeBytes:
				tb.ExporterQueueBatchSendSizeBytes.Record(ctx, value, senderAttr)
			case queuebatchtelemetry.MetricInFlight:
				tb.ExporterInFlightRequests.Add(ctx, value, queueAttr)
			case queuebatchtelemetry.MetricSent:
				itemsSentInst.Add(ctx, value, senderAttr)
			case queuebatchtelemetry.MetricSendFailure:
				itemsFailedInst.Add(ctx, value, append([]metric.AddOption{senderAttr}, options...)...)
			}
		},
		RegisterIntFunc: func(m queuebatchtelemetry.Metric, value func() int64) error {
			var err error
			switch m {
			case queuebatchtelemetry.MetricQueueSize:
				err = tb.RegisterExporterQueueSizeCallback(func(_ context.Context, o metric.Int64Observer) error {
					o.Observe(value(), queueAttr)
					return nil
				})
			case queuebatchtelemetry.MetricQueueCapacity:
				err = tb.RegisterExporterQueueCapacityCallback(func(_ context.Context, o metric.Int64Observer) error {
					o.Observe(value(), queueAttr)
					return nil
				})
			case queuebatchtelemetry.MetricPartitionCacheSize:
				err = tb.RegisterExporterQueueBatchPartitionCacheSizeCallback(func(_ context.Context, o metric.Int64Observer) error {
					o.Observe(value(), queueAttr)
					return nil
				})
			case queuebatchtelemetry.MetricPartitionCacheCapacity:
				err = tb.RegisterExporterQueueBatchPartitionCacheCapacityCallback(func(_ context.Context, o metric.Int64Observer) error {
					o.Observe(value(), queueAttr)
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
