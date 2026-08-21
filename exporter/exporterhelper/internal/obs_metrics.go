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
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

// ObsMetrics reports the metrics produced by exporterhelper for one signal; use NewObsMetrics.
type ObsMetrics interface {
	queuebatch.QueueBatchMetrics

	// RecordInFlight counts a change in the number of requests being sent.
	RecordInFlight(ctx context.Context, delta int64)
	// RecordSent counts the items successfully sent.
	RecordSent(ctx context.Context, items int64)
	// RecordSendFailure counts the items that failed to send.
	RecordSendFailure(ctx context.Context, items int64, options ...metric.AddOption)

	// Shutdown releases the resources backing the instruments.
	Shutdown()
}

// RecordInFlightFunc records a change in the requests being sent, +1 at start and -1 at end.
type RecordInFlightFunc func(ctx context.Context, delta int64)

func (f RecordInFlightFunc) RecordInFlight(ctx context.Context, delta int64) {
	if f == nil {
		return
	}
	f(ctx, delta)
}

// RecordSentFunc records the number of items successfully sent.
type RecordSentFunc func(ctx context.Context, items int64)

func (f RecordSentFunc) RecordSent(ctx context.Context, items int64) {
	if f == nil {
		return
	}
	f(ctx, items)
}

// RecordSendFailureFunc records the items that failed to send, with the failure attributes.
type RecordSendFailureFunc func(ctx context.Context, items int64, options ...metric.AddOption)

func (f RecordSendFailureFunc) RecordSendFailure(ctx context.Context, items int64, options ...metric.AddOption) {
	if f == nil {
		return
	}
	f(ctx, items, options...)
}

// ShutdownFunc releases the resources backing the instruments.
type ShutdownFunc func()

func (f ShutdownFunc) Shutdown() {
	if f == nil {
		return
	}
	f()
}

// ObsMetricsOption configures ObsMetrics.
type ObsMetricsOption interface {
	applyObsMetrics(*obsMetrics)
}

type obsMetricsOptionFunc func(*obsMetrics)

func (f obsMetricsOptionFunc) applyObsMetrics(metrics *obsMetrics) {
	f(metrics)
}

// WithQueueBatchMetrics configures the queue and batch metrics.
func WithQueueBatchMetrics(metrics queuebatch.QueueBatchMetrics) ObsMetricsOption {
	return obsMetricsOptionFunc(func(obs *obsMetrics) {
		if metrics != nil {
			obs.QueueBatchMetrics = metrics
		}
	})
}

// WithRecordInFlight configures how in-flight request changes are recorded.
func WithRecordInFlight(record RecordInFlightFunc) ObsMetricsOption {
	return obsMetricsOptionFunc(func(metrics *obsMetrics) {
		metrics.RecordInFlightFunc = record
	})
}

// WithRecordSent configures how successfully sent items are recorded.
func WithRecordSent(record RecordSentFunc) ObsMetricsOption {
	return obsMetricsOptionFunc(func(metrics *obsMetrics) {
		metrics.RecordSentFunc = record
	})
}

// WithRecordSendFailure configures how send failures are recorded.
func WithRecordSendFailure(record RecordSendFailureFunc) ObsMetricsOption {
	return obsMetricsOptionFunc(func(metrics *obsMetrics) {
		metrics.RecordSendFailureFunc = record
	})
}

// WithMetricsShutdown configures how metric resources are released.
func WithMetricsShutdown(shutdown ShutdownFunc) ObsMetricsOption {
	return obsMetricsOptionFunc(func(metrics *obsMetrics) {
		metrics.ShutdownFunc = shutdown
	})
}

// NewObsMetrics returns ObsMetrics whose unspecified operations report nothing.
func NewObsMetrics(options ...ObsMetricsOption) ObsMetrics {
	metrics := obsMetrics{
		QueueBatchMetrics: queuebatch.NewQueueBatchMetrics(),
	}
	for _, option := range options {
		option.applyObsMetrics(&metrics)
	}
	return metrics
}

// obsMetrics implements ObsMetrics by extending a QueueBatchMetrics.
type obsMetrics struct {
	queuebatch.QueueBatchMetrics
	RecordInFlightFunc
	RecordSentFunc
	RecordSendFailureFunc
	ShutdownFunc
}

var _ ObsMetrics = obsMetrics{}

// recordSize records the item count, and the byte size when bytesInst is enabled.
func recordSize(inst, bytesInst metric.Int64Histogram, attrs metric.MeasurementOption) func(context.Context, int64, queue.Int64Value) {
	return func(ctx context.Context, items int64, bytesSize queue.Int64Value) {
		inst.Record(ctx, items, attrs)
		if bytesInst.Enabled(ctx) {
			bytesInst.Record(ctx, bytesSize.Value(), attrs)
		}
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

	// The signal is an attribute only for instruments whose name omits it.
	enqueueFailedAttrs := attrs(exporterAttr)
	queueAttrs := attrs(exporterAttr, signalAttr)
	batchAttrs := destAttrs(exporterAttr, signalAttr)
	sentAttrs := destAttrs(exporterAttr)
	inFlightAttrs := attrs(exporterAttr, signalAttr)

	var itemsSentInst, itemsFailedInst, enqueueFailedInst metric.Int64Counter
	switch signal {
	case pipeline.SignalTraces:
		itemsSentInst = tb.ExporterSentSpans
		itemsFailedInst = tb.ExporterSendFailedSpans
		enqueueFailedInst = tb.ExporterEnqueueFailedSpans
	case pipeline.SignalMetrics:
		itemsSentInst = tb.ExporterSentMetricPoints
		itemsFailedInst = tb.ExporterSendFailedMetricPoints
		enqueueFailedInst = tb.ExporterEnqueueFailedMetricPoints
	case pipeline.SignalLogs:
		itemsSentInst = tb.ExporterSentLogRecords
		itemsFailedInst = tb.ExporterSendFailedLogRecords
		enqueueFailedInst = tb.ExporterEnqueueFailedLogRecords
	case xpipeline.SignalProfiles:
		itemsSentInst = tb.ExporterSentProfileSamples
		itemsFailedInst = tb.ExporterSendFailedProfileSamples
		enqueueFailedInst = tb.ExporterEnqueueFailedProfileSamples
	}

	// The instruments above are nil for unknown signals, whose operations go unreported.
	var recordEnqueueFailure queue.RecordEnqueueFailureFunc
	if enqueueFailedInst != nil {
		recordEnqueueFailure = func(ctx context.Context, items int64) {
			enqueueFailedInst.Add(ctx, items, enqueueFailedAttrs)
		}
	}
	var recordSent RecordSentFunc
	if itemsSentInst != nil {
		recordSent = func(ctx context.Context, items int64) {
			itemsSentInst.Add(ctx, items, sentAttrs)
		}
	}
	var recordSendFailure RecordSendFailureFunc
	if itemsFailedInst != nil {
		recordSendFailure = func(ctx context.Context, items int64, options ...metric.AddOption) {
			itemsFailedInst.Add(ctx, items, append([]metric.AddOption{sentAttrs}, options...)...)
		}
	}

	return NewObsMetrics(
		WithQueueBatchMetrics(queuebatch.NewQueueBatchMetrics(
			queuebatch.WithQueueMetrics(queue.NewQueueMetrics(
				queue.WithRecordEnqueueFailure(recordEnqueueFailure),
				queue.WithRecordEnqueueSize(recordSize(tb.ExporterEnqueueSize, tb.ExporterEnqueueSizeBytes, queueAttrs)),
				queue.WithRegisterQueueSize(func(observeSize queue.Int64Value) error {
					return tb.RegisterExporterQueueSizeCallback(func(_ context.Context, o metric.Int64Observer) error {
						o.Observe(observeSize.Value(), queueAttrs)
						return nil
					})
				}),
				queue.WithRegisterQueueCapacity(func(observeCapacity queue.Int64Value) error {
					return tb.RegisterExporterQueueCapacityCallback(func(_ context.Context, o metric.Int64Observer) error {
						o.Observe(observeCapacity.Value(), queueAttrs)
						return nil
					})
				}),
			)),
			queuebatch.WithRecordBatchSendSize(recordSize(
				tb.ExporterQueueBatchSendSize,
				tb.ExporterQueueBatchSendSizeBytes,
				batchAttrs,
			)),
		)),
		WithRecordInFlight(func(ctx context.Context, delta int64) {
			tb.ExporterInFlightRequests.Add(ctx, delta, inFlightAttrs)
		}),
		WithRecordSent(recordSent),
		WithRecordSendFailure(recordSendFailure),
		WithMetricsShutdown(tb.Shutdown),
	), nil
}
