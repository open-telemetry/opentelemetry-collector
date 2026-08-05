// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
)

// QueueObserver observes the current size or capacity of a queue. Exporterhelper
// passes an observer to the RegisterQueueSize and RegisterQueueCapacity
// operations, which are expected to install it as an asynchronous callback.
//
// Experimental: This API is at the early stage of development and may change
// without backward compatibility until
// https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is
// resolved.
type QueueObserver = queue.QueueObserver

// ObsMetricsConfig defines the operations invoked by exporterhelper to report
// its observation events. Exporterhelper owns the meaning and timing of each
// operation; a component supplies the instruments. Nil operations are no-ops,
// so a component only implements the events it reports.
//
// Experimental: This API is at the early stage of development and may change
// without backward compatibility until
// https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is
// resolved.
type ObsMetricsConfig struct {
	// RecordEnqueueFailure reports the number of items dropped because they
	// could not be added to the queue.
	RecordEnqueueFailure func(ctx context.Context, items int64)

	// RecordBatchSendSize reports the number of items and bytes in a request
	// as it is offered to the queue.
	RecordBatchSendSize func(ctx context.Context, items, bytes int64)

	// RegisterQueueSize installs an observer for the current queue size.
	RegisterQueueSize func(observe QueueObserver) error

	// RegisterQueueCapacity installs an observer for the fixed queue capacity.
	RegisterQueueCapacity func(observe QueueObserver) error

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
//
// Experimental: This API is at the early stage of development and may change
// without backward compatibility until
// https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is
// resolved.
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

func (m *ObsMetrics) RegisterQueueSize(observe QueueObserver) error {
	if m.config.RegisterQueueSize == nil {
		return nil
	}
	return m.config.RegisterQueueSize(observe)
}

func (m *ObsMetrics) RegisterQueueCapacity(observe QueueObserver) error {
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
