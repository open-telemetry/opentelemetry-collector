// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"
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
) (queuebatchtelemetry.QueueMetrics, error) {
	tb, err := metadata.NewTelemetryBuilder(telemetry)
	if err != nil {
		return queuebatchtelemetry.QueueMetrics{}, err
	}

	exporterAttr := attribute.String(exporterKey, id.String())
	metricAttr := metric.WithAttributeSet(attribute.NewSet(exporterAttr))
	queueAttr := metric.WithAttributeSet(attribute.NewSet(
		exporterAttr,
		attribute.String(dataTypeKey, signal.String()),
	))
	shutdown := sync.OnceFunc(tb.Shutdown)

	var enqueueFailedInst metric.Int64Counter
	switch signal {
	case pipeline.SignalTraces:
		enqueueFailedInst = tb.ExporterEnqueueFailedSpans
	case pipeline.SignalMetrics:
		enqueueFailedInst = tb.ExporterEnqueueFailedMetricPoints
	case pipeline.SignalLogs:
		enqueueFailedInst = tb.ExporterEnqueueFailedLogRecords
	case xpipeline.SignalProfiles:
		enqueueFailedInst = tb.ExporterEnqueueFailedProfileSamples
	}

	return queuebatchtelemetry.QueueMetrics{
		EnqueueFailureFunc: func(ctx context.Context, items int64) {
			enqueueFailedInst.Add(ctx, items, metricAttr)
		},
		EnqueueSizeFunc: func(ctx context.Context, items int64, bytesSize func() int64) {
			tb.ExporterEnqueueSize.Record(ctx, items, metricAttr)
			if tb.ExporterEnqueueSizeBytes.Enabled(ctx) {
				tb.ExporterEnqueueSizeBytes.Record(ctx, bytesSize(), metricAttr)
			}
		},
		RegisterQueueFunc: func(size, capacity func() int64) error {
			if err := tb.RegisterExporterQueueSizeCallback(func(_ context.Context, o metric.Int64Observer) error {
				o.Observe(size(), queueAttr)
				return nil
			}); err != nil {
				shutdown()
				return err
			}
			if err := tb.RegisterExporterQueueCapacityCallback(func(_ context.Context, o metric.Int64Observer) error {
				o.Observe(capacity(), queueAttr)
				return nil
			}); err != nil {
				shutdown()
				return err
			}
			return nil
		},
		ShutdownFunc: shutdown,
	}, nil
}
