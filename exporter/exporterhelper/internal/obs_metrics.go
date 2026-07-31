// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

// ObsMetrics reports the metrics produced by exporterhelper for one signal.
// Components that reuse exporterhelper can provide their own implementation to
// report metrics using names and attributes appropriate for that component.
// Exporterhelper owns the instance after WithObsMetrics applies successfully
// and calls Shutdown when construction fails or the component shuts down.
type ObsMetrics interface {
	queue.ObsMetrics
	RecordInFlight(context.Context, int64)
	RecordSent(context.Context, int64)
	RecordSendFailure(context.Context, int64, ...metric.AddOption)
	Shutdown()
}

type exporterObsMetrics struct {
	tb                     *metadata.TelemetryBuilder
	metricAttr             metric.MeasurementOption
	queueMetricAttr        metric.MeasurementOption
	inFlightMetricAttr     metric.MeasurementOption
	asyncAttr              metric.MeasurementOption
	enqueueFailedInst      metric.Int64Counter
	itemsSentInst          metric.Int64Counter
	itemsFailedInst        metric.Int64Counter
	inFlightInst           metric.Int64UpDownCounter
	queueBatchSizeInst     metric.Int64Histogram
	queueBatchSizeByteInst metric.Int64Histogram
}

func newExporterObsMetrics(
	tel component.TelemetrySettings,
	id component.ID,
	signal pipeline.Signal,
	extraAttrs []attribute.KeyValue,
) (ObsMetrics, error) {
	tb, err := metadata.NewTelemetryBuilder(tel)
	if err != nil {
		return nil, err
	}

	exporterAttr := attribute.String(ExporterKey, id.String())
	om := &exporterObsMetrics{
		tb:                     tb,
		metricAttr:             metric.WithAttributeSet(attribute.NewSet(append(extraAttrs, exporterAttr)...)),
		queueMetricAttr:        metric.WithAttributeSet(attribute.NewSet(exporterAttr)),
		inFlightMetricAttr:     metric.WithAttributeSet(attribute.NewSet(exporterAttr, attribute.String(DataTypeKey, signal.String()))),
		asyncAttr:              metric.WithAttributeSet(attribute.NewSet(exporterAttr, attribute.String(DataTypeKey, signal.String()))),
		inFlightInst:           tb.ExporterInFlightRequests,
		queueBatchSizeInst:     tb.ExporterQueueBatchSendSize,
		queueBatchSizeByteInst: tb.ExporterQueueBatchSendSizeBytes,
	}

	switch signal {
	case pipeline.SignalTraces:
		om.enqueueFailedInst = tb.ExporterEnqueueFailedSpans
		om.itemsSentInst = tb.ExporterSentSpans
		om.itemsFailedInst = tb.ExporterSendFailedSpans
	case pipeline.SignalMetrics:
		om.enqueueFailedInst = tb.ExporterEnqueueFailedMetricPoints
		om.itemsSentInst = tb.ExporterSentMetricPoints
		om.itemsFailedInst = tb.ExporterSendFailedMetricPoints
	case pipeline.SignalLogs:
		om.enqueueFailedInst = tb.ExporterEnqueueFailedLogRecords
		om.itemsSentInst = tb.ExporterSentLogRecords
		om.itemsFailedInst = tb.ExporterSendFailedLogRecords
	case xpipeline.SignalProfiles:
		om.enqueueFailedInst = tb.ExporterEnqueueFailedProfileSamples
		om.itemsSentInst = tb.ExporterSentProfileSamples
		om.itemsFailedInst = tb.ExporterSendFailedProfileSamples
	}

	return om, nil
}

func (om *exporterObsMetrics) RecordEnqueueFailure(ctx context.Context, items int64) {
	if om.enqueueFailedInst != nil {
		om.enqueueFailedInst.Add(ctx, items, om.queueMetricAttr)
	}
}

func (om *exporterObsMetrics) RecordBatchSendSize(ctx context.Context, items, bytes int64) {
	om.queueBatchSizeInst.Record(ctx, items, om.queueMetricAttr)
	om.queueBatchSizeByteInst.Record(ctx, bytes, om.queueMetricAttr)
}

func (om *exporterObsMetrics) RegisterQueueSize(observe func() int64) error {
	return om.tb.RegisterExporterQueueSizeCallback(func(_ context.Context, o metric.Int64Observer) error {
		o.Observe(observe(), om.asyncAttr)
		return nil
	})
}

func (om *exporterObsMetrics) RegisterQueueCapacity(observe func() int64) error {
	return om.tb.RegisterExporterQueueCapacityCallback(func(_ context.Context, o metric.Int64Observer) error {
		o.Observe(observe(), om.asyncAttr)
		return nil
	})
}

func (om *exporterObsMetrics) RecordInFlight(ctx context.Context, delta int64) {
	if om.inFlightInst != nil {
		om.inFlightInst.Add(ctx, delta, om.inFlightMetricAttr)
	}
}

func (om *exporterObsMetrics) RecordSent(ctx context.Context, items int64) {
	if om.itemsSentInst != nil {
		om.itemsSentInst.Add(ctx, items, om.metricAttr)
	}
}

func (om *exporterObsMetrics) RecordSendFailure(ctx context.Context, items int64, options ...metric.AddOption) {
	if om.itemsFailedInst != nil {
		om.itemsFailedInst.Add(ctx, items, append([]metric.AddOption{om.metricAttr}, options...)...)
	}
}

func (om *exporterObsMetrics) Shutdown() {
	om.tb.Shutdown()
}
