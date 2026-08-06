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
	queue.QueueBatchMetrics
	RecordInFlight(context.Context, int64)
	RecordSent(context.Context, int64)
	RecordSendFailure(context.Context, int64, ...metric.AddOption)
	Shutdown()
}

type RecordInFlightFunc func(context.Context, int64)

func (f RecordInFlightFunc) RecordInFlight(ctx context.Context, delta int64) {
	if f != nil {
		f(ctx, delta)
	}
}

type RecordSentFunc func(context.Context, int64)

func (f RecordSentFunc) RecordSent(ctx context.Context, items int64) {
	if f != nil {
		f(ctx, items)
	}
}

type RecordSendFailureFunc func(context.Context, int64, ...metric.AddOption)

func (f RecordSendFailureFunc) RecordSendFailure(ctx context.Context, items int64, options ...metric.AddOption) {
	if f != nil {
		f(ctx, items, options...)
	}
}

type ShutdownObsMetricsFunc func()

func (f ShutdownObsMetricsFunc) Shutdown() {
	if f != nil {
		f()
	}
}

type exporterObsMetrics struct {
	queue.RecordEnqueueFailureFunc
	queue.RecordBatchSendSizeFunc
	queue.RegisterQueueSizeFunc
	queue.RegisterQueueCapacityFunc
	RecordInFlightFunc
	RecordSentFunc
	RecordSendFailureFunc
	ShutdownObsMetricsFunc
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
	queueAttr := metric.WithAttributeSet(attribute.NewSet(exporterAttr))
	senderAttr := metric.WithAttributeSet(attribute.NewSet(append(extraAttrs, exporterAttr)...))
	asyncAttr := metric.WithAttributeSet(attribute.NewSet(exporterAttr, attribute.String(DataTypeKey, signal.String())))
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
	om := &exporterObsMetrics{
		RecordBatchSendSizeFunc: func(ctx context.Context, items, bytes int64) {
			tb.ExporterQueueBatchSendSize.Record(ctx, items, queueAttr)
			tb.ExporterQueueBatchSendSizeBytes.Record(ctx, bytes, queueAttr)
		},
		RegisterQueueSizeFunc: func(observe queue.QueueObserver) error {
			return tb.RegisterExporterQueueSizeCallback(func(_ context.Context, o metric.Int64Observer) error {
				o.Observe(observe.Observe(), asyncAttr)
				return nil
			})
		},
		RegisterQueueCapacityFunc: func(observe queue.QueueObserver) error {
			return tb.RegisterExporterQueueCapacityCallback(func(_ context.Context, o metric.Int64Observer) error {
				o.Observe(observe.Observe(), asyncAttr)
				return nil
			})
		},
		RecordInFlightFunc: func(ctx context.Context, delta int64) {
			tb.ExporterInFlightRequests.Add(ctx, delta, asyncAttr)
		},
		ShutdownObsMetricsFunc: tb.Shutdown,
	}
	// The instruments below are nil only when the signal is not one of the
	// known signals, in which case those events go unreported.
	if enqueueFailedInst != nil {
		om.RecordEnqueueFailureFunc = func(ctx context.Context, items int64) {
			enqueueFailedInst.Add(ctx, items, queueAttr)
		}
	}
	if itemsSentInst != nil {
		om.RecordSentFunc = func(ctx context.Context, items int64) {
			itemsSentInst.Add(ctx, items, senderAttr)
		}
	}
	if itemsFailedInst != nil {
		om.RecordSendFailureFunc = func(ctx context.Context, items int64, options ...metric.AddOption) {
			itemsFailedInst.Add(ctx, items, append([]metric.AddOption{senderAttr}, options...)...)
		}
	}

	return om, nil
}
