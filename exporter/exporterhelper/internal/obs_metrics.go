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
// Exporterhelper owns the meaning and timing of each operation; the
// implementation supplies the instruments. Components that reuse
// exporterhelper implement this to report metrics using names and attributes
// appropriate for that component, most easily by setting the operations of a
// FuncObsMetrics.
type QueueBatchMetrics interface {
	queue.QueueBatchMetrics

	RecordInFlight(ctx context.Context, delta int64)
	RecordSent(ctx context.Context, items int64)
	RecordSendFailure(ctx context.Context, items int64, options ...metric.AddOption)

	// Shutdown releases the resources backing the instruments. Exporterhelper
	// calls it when the component shuts down, and only takes on that
	// responsibility once its constructor returns successfully.
	Shutdown()
}

// RecordInFlightFunc records a change in the number of requests currently
// being sent. Delta is +1 when a send starts and -1 when it ends.
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

// RecordSendFailureFunc records the number of items that failed to send. The
// options carry the failure attributes derived from the error.
type RecordSendFailureFunc func(ctx context.Context, items int64, options ...metric.AddOption)

func (f RecordSendFailureFunc) RecordSendFailure(ctx context.Context, items int64, options ...metric.AddOption) {
	if f == nil {
		return
	}
	f(ctx, items, options...)
}

// ShutdownObsMetricsFunc releases the resources backing the instruments.
type ShutdownObsMetricsFunc func()

func (f ShutdownObsMetricsFunc) Shutdown() {
	if f == nil {
		return
	}
	f()
}

// obsMetrics implements ObsMetrics from a set of operations. The zero
// value reports nothing, so a component only sets the events it reports.
type FuncObsMetrics struct {
	queue.QueueBatchMetrics
	RecordInFlightFunc
	RecordSentFunc
	RecordSendFailureFunc
	ShutdownObsMetricsFunc
}

// newExporterObsMetrics reports through the exporter-oriented instruments,
// which is what exporterhelper uses unless a component overrides them.
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

	om := &obsMetrics{
		RecordEnqueueItemsFunc: func(ctx context.Context, items, bytes int64) {
			tb.ExporterQueueBatchSendSize.Record(ctx, items, queueAttr)
			tb.ExporterQueueBatchSendSizeBytes.Record(ctx, bytes, queueAttr)
		},
		RegisterQueueSizeFunc: func(observeSize func() int64) error {
			return tb.RegisterExporterQueueSizeCallback(func(_ context.Context, o metric.Int64Observer) error {
				o.Observe(observeSize(), asyncAttr)
				return nil
			})
		},
		RegisterQueueCapacityFunc: func(observeCapacity func() int64) error {
			return tb.RegisterExporterQueueCapacityCallback(func(_ context.Context, o metric.Int64Observer) error {
				o.Observe(observeCapacity(), asyncAttr)
				return nil
			})
		},
		RecordInFlightFunc: func(ctx context.Context, delta int64) {
			tb.ExporterInFlightRequests.Add(ctx, delta, asyncAttr)
		},
		ShutdownObsMetricsFunc: tb.Shutdown,
	}

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
