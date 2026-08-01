// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
)

// QueueObserver observes the current size or capacity of a queue.
type QueueObserver = queue.QueueObserver

// ObserveQueueFunc returns the current size or capacity of a queue.
type ObserveQueueFunc = queue.ObserveQueueFunc

type (
	RecordEnqueueFailureFunc  = queue.RecordEnqueueFailureFunc
	RecordBatchSendSizeFunc   = queue.RecordBatchSendSizeFunc
	RegisterQueueSizeFunc     = queue.RegisterQueueSizeFunc
	RegisterQueueCapacityFunc = queue.RegisterQueueCapacityFunc
	RecordInFlightFunc        = internal.RecordInFlightFunc
	RecordSentFunc            = internal.RecordSentFunc
	RecordSendFailureFunc     = internal.RecordSendFailureFunc
	ShutdownObsMetricsFunc    = internal.ShutdownObsMetricsFunc
)

// Config defines the operation functions used by ObsMetrics. Nil functions are
// no-ops.
type Config struct {
	RecordEnqueueFailure  RecordEnqueueFailureFunc
	RecordBatchSendSize   RecordBatchSendSizeFunc
	RegisterQueueSize     RegisterQueueSizeFunc
	RegisterQueueCapacity RegisterQueueCapacityFunc
	RecordInFlight        RecordInFlightFunc
	RecordSent            RecordSentFunc
	RecordSendFailure     RecordSendFailureFunc
	Shutdown              ShutdownObsMetricsFunc

	// prevent unkeyed literal initialization
	_ struct{}
}

// ObsMetrics reports the metrics produced by exporterhelper for one signal.
type ObsMetrics struct {
	config Config
}

// ObsMetricsOption applies an option to ObsMetrics.
type ObsMetricsOption interface {
	apply(*ObsMetrics)
}

type obsMetricsOptionFunc func(*ObsMetrics)

func (f obsMetricsOptionFunc) apply(metrics *ObsMetrics) {
	f(metrics)
}

// NewObsMetrics creates ObsMetrics from operation functions. Exporterhelper
// defines the meaning and timing of each operation; unset operations are
// no-ops.
func NewObsMetrics(options ...ObsMetricsOption) *ObsMetrics {
	metrics := &ObsMetrics{}
	for _, option := range options {
		option.apply(metrics)
	}
	return metrics
}

// WithConfig sets the operation functions used by ObsMetrics.
func WithConfig(cfg Config) ObsMetricsOption {
	return obsMetricsOptionFunc(func(metrics *ObsMetrics) {
		metrics.config = cfg
	})
}

func (m *ObsMetrics) RecordEnqueueFailure(ctx context.Context, items int64) {
	m.config.RecordEnqueueFailure.RecordEnqueueFailure(ctx, items)
}

func (m *ObsMetrics) RecordBatchSendSize(ctx context.Context, items, bytes int64) {
	m.config.RecordBatchSendSize.RecordBatchSendSize(ctx, items, bytes)
}

func (m *ObsMetrics) RegisterQueueSize(observe QueueObserver) error {
	return m.config.RegisterQueueSize.RegisterQueueSize(observe)
}

func (m *ObsMetrics) RegisterQueueCapacity(observe QueueObserver) error {
	return m.config.RegisterQueueCapacity.RegisterQueueCapacity(observe)
}

func (m *ObsMetrics) RecordInFlight(ctx context.Context, delta int64) {
	m.config.RecordInFlight.RecordInFlight(ctx, delta)
}

func (m *ObsMetrics) RecordSent(ctx context.Context, items int64) {
	m.config.RecordSent.RecordSent(ctx, items)
}

func (m *ObsMetrics) RecordSendFailure(ctx context.Context, items int64, options ...metric.AddOption) {
	m.config.RecordSendFailure.RecordSendFailure(ctx, items, options...)
}

func (m *ObsMetrics) Shutdown() {
	m.config.Shutdown.Shutdown()
}
