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
	signalKey    = "otel.signal"
)

type obsMetrics struct {
	tb                     *metadata.TelemetryBuilder
	metricAttr             metric.MeasurementOption
	enqueueFailedInst      metric.Int64Counter
	itemsSentInst          metric.Int64Counter
	itemsFailedInst        metric.Int64Counter
	inFlightInst           metric.Int64UpDownCounter
	queueBatchSizeInst     metric.Int64Histogram
	queueBatchSizeByteInst metric.Int64Histogram
}

// newObsMetrics reports the exporterhelper observation events through
// processor-oriented instruments.
func newObsMetrics(set component.TelemetrySettings, id component.ID, signal pipeline.Signal) (exporterhelper.ObsMetrics, error) {
	tb, err := metadata.NewTelemetryBuilder(set)
	if err != nil {
		return nil, err
	}

	processorAttr := attribute.String(processorKey, id.String())
	om := &obsMetrics{
		tb:                     tb,
		metricAttr:             metric.WithAttributeSet(attribute.NewSet(processorAttr, attribute.String(signalKey, signal.String()))),
		enqueueFailedInst:      tb.ProcessorQueuebatchEnqueueFailedItems,
		itemsSentInst:          tb.ProcessorQueuebatchSentItems,
		itemsFailedInst:        tb.ProcessorQueuebatchSendFailedItems,
		inFlightInst:           tb.ProcessorQueuebatchInFlightRequests,
		queueBatchSizeInst:     tb.ProcessorQueuebatchBatchSendSize,
		queueBatchSizeByteInst: tb.ProcessorQueuebatchBatchSendSizeBytes,
	}
	return om, nil
}

func (om *obsMetrics) RecordEnqueueFailure(ctx context.Context, items int64) {
	om.enqueueFailedInst.Add(ctx, items, om.metricAttr)
}

func (om *obsMetrics) RecordEnqueueItems(ctx context.Context, items, bytes int64) {
	om.queueBatchSizeInst.Record(ctx, items, om.metricAttr)
	om.queueBatchSizeByteInst.Record(ctx, bytes, om.metricAttr)
}

func (om *obsMetrics) RegisterQueueSize(observeSize func() int64) error {
	return om.tb.RegisterProcessorQueuebatchQueueSizeCallback(func(_ context.Context, o metric.Int64Observer) error {
		o.Observe(observeSize(), om.metricAttr)
		return nil
	})
}

func (om *obsMetrics) RegisterQueueCapacity(observeCapacity func() int64) error {
	return om.tb.RegisterProcessorQueuebatchQueueCapacityCallback(func(_ context.Context, o metric.Int64Observer) error {
		o.Observe(observeCapacity(), om.metricAttr)
		return nil
	})
}

func (om *obsMetrics) RecordInFlight(ctx context.Context, delta int64) {
	om.inFlightInst.Add(ctx, delta, om.metricAttr)
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
