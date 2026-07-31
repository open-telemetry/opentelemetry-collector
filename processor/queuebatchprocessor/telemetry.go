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
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/processor/queuebatchprocessor/internal/metadata"
)

const (
	processorKey = "processor"
	dataTypeKey  = "data_type"
)

type obsMetrics struct {
	tb                     *metadata.TelemetryBuilder
	metricAttr             metric.MeasurementOption
	asyncAttr              metric.MeasurementOption
	enqueueFailedInst      metric.Int64Counter
	itemsSentInst          metric.Int64Counter
	itemsFailedInst        metric.Int64Counter
	inFlightInst           metric.Int64UpDownCounter
	queueBatchSizeInst     metric.Int64Histogram
	queueBatchSizeByteInst metric.Int64Histogram
}

func newObsMetrics(set component.TelemetrySettings, id component.ID, signal pipeline.Signal) (exporterhelper.ObsMetrics, error) {
	tb, err := metadata.NewTelemetryBuilder(set)
	if err != nil {
		return nil, err
	}

	processorAttr := attribute.String(processorKey, id.String())
	om := &obsMetrics{
		tb:                     tb,
		metricAttr:             metric.WithAttributeSet(attribute.NewSet(processorAttr)),
		asyncAttr:              metric.WithAttributeSet(attribute.NewSet(processorAttr, attribute.String(dataTypeKey, signal.String()))),
		inFlightInst:           tb.ProcessorInFlightRequests,
		queueBatchSizeInst:     tb.ProcessorQueueBatchSendSize,
		queueBatchSizeByteInst: tb.ProcessorQueueBatchSendSizeBytes,
	}

	switch signal {
	case pipeline.SignalTraces:
		om.enqueueFailedInst = tb.ProcessorEnqueueFailedSpans
		om.itemsSentInst = tb.ProcessorSentSpans
		om.itemsFailedInst = tb.ProcessorSendFailedSpans
	case pipeline.SignalMetrics:
		om.enqueueFailedInst = tb.ProcessorEnqueueFailedMetricPoints
		om.itemsSentInst = tb.ProcessorSentMetricPoints
		om.itemsFailedInst = tb.ProcessorSendFailedMetricPoints
	case pipeline.SignalLogs:
		om.enqueueFailedInst = tb.ProcessorEnqueueFailedLogRecords
		om.itemsSentInst = tb.ProcessorSentLogRecords
		om.itemsFailedInst = tb.ProcessorSendFailedLogRecords
	case xpipeline.SignalProfiles:
		om.enqueueFailedInst = tb.ProcessorEnqueueFailedProfileSamples
		om.itemsSentInst = tb.ProcessorSentProfileSamples
		om.itemsFailedInst = tb.ProcessorSendFailedProfileSamples
	}
	return exporterhelper.NewObsMetrics(
		exporterhelper.WithConfig(exporterhelper.Config{
			RecordEnqueueFailure:  om.RecordEnqueueFailure,
			RecordBatchSendSize:   om.RecordBatchSendSize,
			RegisterQueueSize:     om.RegisterQueueSize,
			RegisterQueueCapacity: om.RegisterQueueCapacity,
			RecordInFlight:        om.RecordInFlight,
			RecordSent:            om.RecordSent,
			RecordSendFailure:     om.RecordSendFailure,
			Shutdown:              om.Shutdown,
		}),
	), nil
}

func (om *obsMetrics) RecordEnqueueFailure(ctx context.Context, items int64) {
	om.enqueueFailedInst.Add(ctx, items, om.metricAttr)
}

func (om *obsMetrics) RecordBatchSendSize(ctx context.Context, items, bytes int64) {
	om.queueBatchSizeInst.Record(ctx, items, om.metricAttr)
	om.queueBatchSizeByteInst.Record(ctx, bytes, om.metricAttr)
}

func (om *obsMetrics) RegisterQueueSize(observe exporterhelper.QueueObserver) error {
	return om.tb.RegisterProcessorQueueSizeCallback(func(_ context.Context, o metric.Int64Observer) error {
		o.Observe(observe.Observe(), om.asyncAttr)
		return nil
	})
}

func (om *obsMetrics) RegisterQueueCapacity(observe exporterhelper.QueueObserver) error {
	return om.tb.RegisterProcessorQueueCapacityCallback(func(_ context.Context, o metric.Int64Observer) error {
		o.Observe(observe.Observe(), om.asyncAttr)
		return nil
	})
}

func (om *obsMetrics) RecordInFlight(ctx context.Context, delta int64) {
	om.inFlightInst.Add(ctx, delta, om.asyncAttr)
}

func (om *obsMetrics) RecordSent(ctx context.Context, items int64) {
	om.itemsSentInst.Add(ctx, items, om.metricAttr)
}

func (om *obsMetrics) RecordSendFailure(ctx context.Context, items int64, options ...metric.AddOption) {
	om.itemsFailedInst.Add(ctx, items, append([]metric.AddOption{om.metricAttr}, options...)...)
}

func (om *obsMetrics) Shutdown() {
	om.tb.Shutdown()
}
