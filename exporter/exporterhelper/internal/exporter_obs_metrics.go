// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	queuebatchtelemetry "go.opentelemetry.io/collector/internal/telemetry/queuebatch"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

func newExporterObsMetrics(
	set exporter.Settings,
	signal pipeline.Signal,
	extraAttrs []attribute.KeyValue,
) (ObsMetrics, error) {
	queueMetrics, err := queue.NewExporterObsMetrics(set.TelemetrySettings, set.ID, signal)
	if err != nil {
		return ObsMetrics{}, err
	}

	sendMetrics, sendMetricsShutdown, err := newExporterSendMetrics(set, signal, extraAttrs)
	if err != nil {
		queueMetrics.Shutdown()
		return ObsMetrics{}, err
	}

	obsMetrics := ObsMetrics{
		QueueMetrics: queueMetrics,
		SendMetrics:  sendMetrics,
	}
	obsMetrics.ShutdownFunc = sync.OnceFunc(func() {
		queueMetrics.Shutdown()
		sendMetricsShutdown.Shutdown()
	})
	return obsMetrics, nil
}

func newExporterSendMetrics(
	set exporter.Settings,
	signal pipeline.Signal,
	extraAttrs []attribute.KeyValue,
) (queuebatchtelemetry.SendMetrics, queuebatchtelemetry.ShutdownFunc, error) {
	tb, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return queuebatchtelemetry.SendMetrics{}, nil, err
	}

	exporterAttr := attribute.String(ExporterKey, set.ID.String())
	metricAttr := metric.WithAttributeSet(attribute.NewSet(append(extraAttrs, exporterAttr)...))
	inFlightAttr := metric.WithAttributeSet(attribute.NewSet(
		exporterAttr,
		attribute.String(DataTypeKey, signal.String()),
	))

	var itemsSentInst, itemsFailedInst metric.Int64Counter
	switch signal {
	case pipeline.SignalTraces:
		itemsSentInst = tb.ExporterSentSpans
		itemsFailedInst = tb.ExporterSendFailedSpans
	case pipeline.SignalMetrics:
		itemsSentInst = tb.ExporterSentMetricPoints
		itemsFailedInst = tb.ExporterSendFailedMetricPoints
	case pipeline.SignalLogs:
		itemsSentInst = tb.ExporterSentLogRecords
		itemsFailedInst = tb.ExporterSendFailedLogRecords
	case xpipeline.SignalProfiles:
		itemsSentInst = tb.ExporterSentProfileSamples
		itemsFailedInst = tb.ExporterSendFailedProfileSamples
	}

	sendMetrics := queuebatchtelemetry.SendMetrics{
		BatchSendSizeFunc: func(ctx context.Context, items int64, bytesSize func() int64) {
			tb.ExporterQueueBatchSendSize.Record(ctx, items, metricAttr)
			if tb.ExporterQueueBatchSendSizeBytes.Enabled(ctx) {
				tb.ExporterQueueBatchSendSizeBytes.Record(ctx, bytesSize(), metricAttr)
			}
		},
		InFlightFunc: func(ctx context.Context, delta int64) {
			tb.ExporterInFlightRequests.Add(ctx, delta, inFlightAttr)
		},
		SentFunc: func(ctx context.Context, items int64) {
			itemsSentInst.Add(ctx, items, metricAttr)
		},
		SendFailureFunc: func(ctx context.Context, items int64, options ...metric.AddOption) {
			itemsFailedInst.Add(ctx, items, append([]metric.AddOption{metricAttr}, options...)...)
		},
	}
	return sendMetrics, sync.OnceFunc(tb.Shutdown), nil
}
