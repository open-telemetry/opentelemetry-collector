// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

// ObsMetricsConfig defines the operations invoked by exporterhelper to report
// its observation events. Exporterhelper owns the meaning and timing of each
// operation; a component supplies the instruments. Nil operations are no-ops,
// so a component only implements the events it reports.
type ObsMetricsConfig struct {
	// RecordEnqueueFailure reports the number of items dropped because they
	// could not be added to the queue.
	RecordEnqueueFailure func(ctx context.Context, items int64)

	// RecordBatchSendSize reports the number of items and bytes in a request
	// as it is offered to the queue.
	RecordBatchSendSize func(ctx context.Context, items, bytes int64)

	// RegisterQueueSize installs an observer for the current queue size.
	RegisterQueueSize func(observe queue.QueueObserver) error

	// RegisterQueueCapacity installs an observer for the fixed queue capacity.
	RegisterQueueCapacity func(observe queue.QueueObserver) error

	// RecordInFlight reports a change in the number of requests currently
	// being sent. Delta is +1 when a send starts and -1 when it ends.
	RecordInFlight func(ctx context.Context, delta int64)

	// RecordSent reports the number of items successfully sent.
	RecordSent func(ctx context.Context, items int64)

	// RecordSendFailure reports the number of items that failed to send. The
	// options carry the failure attributes derived from the error.
	RecordSendFailure func(ctx context.Context, items int64, options ...metric.AddOption)

	// Shutdown releases the resources backing the instruments above.
	// ObsMetrics.Shutdown deduplicates calls, so this runs at most once even
	// though both exporterhelper and the component may request it.
	Shutdown func()

	// prevent unkeyed literal initialization
	_ struct{}
}

// ObsMetrics reports the metrics produced by exporterhelper for one signal.
// Components that reuse exporterhelper can supply their own operations to
// report metrics using names and attributes appropriate for that component.
// Exporterhelper owns the instance after WithObsMetrics applies successfully
// and calls Shutdown when construction fails or the component shuts down.
type ObsMetrics struct {
	config       ObsMetricsConfig
	shutdownOnce sync.Once
}

// NewObsMetrics creates ObsMetrics that report through the operations in cfg.
func NewObsMetrics(cfg ObsMetricsConfig) *ObsMetrics {
	return &ObsMetrics{config: cfg}
}

func (m *ObsMetrics) RecordEnqueueFailure(ctx context.Context, items int64) {
	if m.config.RecordEnqueueFailure != nil {
		m.config.RecordEnqueueFailure(ctx, items)
	}
}

func (m *ObsMetrics) RecordBatchSendSize(ctx context.Context, items, bytes int64) {
	if m.config.RecordBatchSendSize != nil {
		m.config.RecordBatchSendSize(ctx, items, bytes)
	}
}

func (m *ObsMetrics) RegisterQueueSize(observe queue.QueueObserver) error {
	if m.config.RegisterQueueSize == nil {
		return nil
	}
	return m.config.RegisterQueueSize(observe)
}

func (m *ObsMetrics) RegisterQueueCapacity(observe queue.QueueObserver) error {
	if m.config.RegisterQueueCapacity == nil {
		return nil
	}
	return m.config.RegisterQueueCapacity(observe)
}

func (m *ObsMetrics) RecordInFlight(ctx context.Context, delta int64) {
	if m.config.RecordInFlight != nil {
		m.config.RecordInFlight(ctx, delta)
	}
}

func (m *ObsMetrics) RecordSent(ctx context.Context, items int64) {
	if m.config.RecordSent != nil {
		m.config.RecordSent(ctx, items)
	}
}

func (m *ObsMetrics) RecordSendFailure(ctx context.Context, items int64, options ...metric.AddOption) {
	if m.config.RecordSendFailure != nil {
		m.config.RecordSendFailure(ctx, items, options...)
	}
}

// Shutdown releases the underlying instruments. It is idempotent, so a caller
// that shuts down after a failed exporter construction cannot double-release
// resources that exporterhelper already released.
func (m *ObsMetrics) Shutdown() {
	m.shutdownOnce.Do(func() {
		if m.config.Shutdown != nil {
			m.config.Shutdown()
		}
	})
}

// newExporterObsMetrics reports through the exporter-oriented instruments,
// which is what exporterhelper uses unless a component overrides them.
func newExporterObsMetrics(
	tel component.TelemetrySettings,
	id component.ID,
	signal pipeline.Signal,
	extraAttrs []attribute.KeyValue,
) (*ObsMetrics, error) {
	tb, err := metadata.NewTelemetryBuilder(tel)
	if err != nil {
		return nil, err
	}

	exporterAttr := attribute.String(ExporterKey, id.String())
	queueAttr := metric.WithAttributeSet(attribute.NewSet(exporterAttr))
	senderAttr := metric.WithAttributeSet(attribute.NewSet(append(extraAttrs, exporterAttr)...))
	asyncAttr := metric.WithAttributeSet(attribute.NewSet(exporterAttr, attribute.String(DataTypeKey, signal.String())))

	cfg := ObsMetricsConfig{
		RecordBatchSendSize: func(ctx context.Context, items, bytes int64) {
			tb.ExporterQueueBatchSendSize.Record(ctx, items, queueAttr)
			tb.ExporterQueueBatchSendSizeBytes.Record(ctx, bytes, queueAttr)
		},
		RegisterQueueSize: func(observe queue.QueueObserver) error {
			return tb.RegisterExporterQueueSizeCallback(func(_ context.Context, o metric.Int64Observer) error {
				o.Observe(observe(), asyncAttr)
				return nil
			})
		},
		RegisterQueueCapacity: func(observe queue.QueueObserver) error {
			return tb.RegisterExporterQueueCapacityCallback(func(_ context.Context, o metric.Int64Observer) error {
				o.Observe(observe(), asyncAttr)
				return nil
			})
		},
		RecordInFlight: func(ctx context.Context, delta int64) {
			tb.ExporterInFlightRequests.Add(ctx, delta, asyncAttr)
		},
		Shutdown: tb.Shutdown,
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
		cfg.RecordEnqueueFailure = func(ctx context.Context, items int64) {
			enqueueFailedInst.Add(ctx, items, queueAttr)
		}
	}
	if itemsSentInst != nil {
		cfg.RecordSent = func(ctx context.Context, items int64) {
			itemsSentInst.Add(ctx, items, senderAttr)
		}
	}
	if itemsFailedInst != nil {
		cfg.RecordSendFailure = func(ctx context.Context, items int64, options ...metric.AddOption) {
			itemsFailedInst.Add(ctx, items, append([]metric.AddOption{senderAttr}, options...)...)
		}
	}

	return NewObsMetrics(cfg), nil
}
