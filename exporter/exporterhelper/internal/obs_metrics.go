// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/obsmetrics"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

type ObsMetrics = obsmetrics.ObsMetrics

type exporterObsMetrics struct {
	tb *metadata.TelemetryBuilder

	enqueueFailedInst metric.Int64Counter
	itemsSentInst     metric.Int64Counter
	itemsFailedInst   metric.Int64Counter

	enqueueFailedAttrs metric.MeasurementOption
	queueAttrs         metric.MeasurementOption
	batchAttrs         metric.MeasurementOption
	sentAttrs          metric.MeasurementOption
	inFlightAttrs      metric.MeasurementOption
}

func (m *exporterObsMetrics) RecordEnqueueFailure(ctx context.Context, items int64) {
	if m.enqueueFailedInst != nil {
		m.enqueueFailedInst.Add(ctx, items, m.enqueueFailedAttrs)
	}
}

func (m *exporterObsMetrics) RecordEnqueueSize(ctx context.Context, items int64, bytesSize func() int64) {
	recordSize(ctx, m.tb.ExporterEnqueueSize, m.tb.ExporterEnqueueSizeBytes, m.queueAttrs, items, bytesSize)
}

func (m *exporterObsMetrics) RegisterQueueSize(observeSize func() int64) error {
	return m.tb.RegisterExporterQueueSizeCallback(func(_ context.Context, o metric.Int64Observer) error {
		o.Observe(observeSize(), m.queueAttrs)
		return nil
	})
}

func (m *exporterObsMetrics) RegisterQueueCapacity(observeCapacity func() int64) error {
	return m.tb.RegisterExporterQueueCapacityCallback(func(_ context.Context, o metric.Int64Observer) error {
		o.Observe(observeCapacity(), m.queueAttrs)
		return nil
	})
}

func (m *exporterObsMetrics) RecordBatchSendSize(ctx context.Context, items int64, bytesSize func() int64) {
	recordSize(ctx, m.tb.ExporterQueueBatchSendSize, m.tb.ExporterQueueBatchSendSizeBytes, m.batchAttrs, items, bytesSize)
}

func (m *exporterObsMetrics) RecordInFlight(ctx context.Context, delta int64) {
	m.tb.ExporterInFlightRequests.Add(ctx, delta, m.inFlightAttrs)
}

func (m *exporterObsMetrics) RecordSent(ctx context.Context, items int64) {
	if m.itemsSentInst != nil {
		m.itemsSentInst.Add(ctx, items, m.sentAttrs)
	}
}

func (m *exporterObsMetrics) RecordSendFailure(ctx context.Context, items int64, options ...metric.AddOption) {
	if m.itemsFailedInst != nil {
		m.itemsFailedInst.Add(ctx, items, append([]metric.AddOption{m.sentAttrs}, options...)...)
	}
}

func (m *exporterObsMetrics) Shutdown() {
	m.tb.Shutdown()
}

func recordSize(
	ctx context.Context,
	inst, bytesInst metric.Int64Histogram,
	attrs metric.MeasurementOption,
	items int64,
	bytesSize func() int64,
) {
	inst.Record(ctx, items, attrs)
	if bytesInst.Enabled(ctx) {
		bytesInst.Record(ctx, bytesSize(), attrs)
	}
}

// newExporterObsMetrics reports through the exporter-oriented instruments.
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

	attrs := func(kvs ...attribute.KeyValue) metric.MeasurementOption {
		return metric.WithAttributeSet(attribute.NewSet(kvs...))
	}
	// Instruments measuring the exchange with the destination also carry the
	// exporter's extra attributes, which describe that destination.
	destAttrs := func(kvs ...attribute.KeyValue) metric.MeasurementOption {
		allAttrs := make([]attribute.KeyValue, 0, len(extraAttrs)+len(kvs))
		allAttrs = append(allAttrs, extraAttrs...)
		allAttrs = append(allAttrs, kvs...)
		return metric.WithAttributeSet(attribute.NewSet(allAttrs...))
	}

	exporterAttr := attribute.String(ExporterKey, id.String())
	signalAttr := attribute.String(DataTypeKey, signal.String())

	m := &exporterObsMetrics{
		tb:                 tb,
		enqueueFailedAttrs: attrs(exporterAttr),
		queueAttrs:         attrs(exporterAttr, signalAttr),
		batchAttrs:         destAttrs(exporterAttr, signalAttr),
		sentAttrs:          destAttrs(exporterAttr),
		inFlightAttrs:      attrs(exporterAttr, signalAttr),
	}

	switch signal {
	case pipeline.SignalTraces:
		m.itemsSentInst = tb.ExporterSentSpans
		m.itemsFailedInst = tb.ExporterSendFailedSpans
		m.enqueueFailedInst = tb.ExporterEnqueueFailedSpans
	case pipeline.SignalMetrics:
		m.itemsSentInst = tb.ExporterSentMetricPoints
		m.itemsFailedInst = tb.ExporterSendFailedMetricPoints
		m.enqueueFailedInst = tb.ExporterEnqueueFailedMetricPoints
	case pipeline.SignalLogs:
		m.itemsSentInst = tb.ExporterSentLogRecords
		m.itemsFailedInst = tb.ExporterSendFailedLogRecords
		m.enqueueFailedInst = tb.ExporterEnqueueFailedLogRecords
	case xpipeline.SignalProfiles:
		m.itemsSentInst = tb.ExporterSentProfileSamples
		m.itemsFailedInst = tb.ExporterSendFailedProfileSamples
		m.enqueueFailedInst = tb.ExporterEnqueueFailedProfileSamples
	}

	return m, nil
}
