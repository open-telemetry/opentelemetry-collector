// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

// ObsMetrics reports the metrics produced by exporterhelper for one signal.
//
// This interface cannot be directly implemented. Implementations must use
// NewObsMetrics.
type ObsMetrics interface {
	RecordEnqueueFailure(context.Context, int64)
	RecordBatchSendSize(context.Context, int64, int64)
	RegisterQueueSize(QueueObserver) error
	RegisterQueueCapacity(QueueObserver) error
	RecordInFlight(context.Context, int64)
	RecordSent(context.Context, int64)
	RecordSendFailure(context.Context, int64, ...metric.AddOption)
	Shutdown()

	unexportedObsMetricsFunc()
}

// QueueObserver observes the current size or capacity of a queue.
type QueueObserver interface {
	Observe() int64
}

// ObserveQueueFunc returns the current size or capacity of a queue.
type ObserveQueueFunc func() int64

var _ QueueObserver = ObserveQueueFunc(nil)

func (f ObserveQueueFunc) Observe() int64 {
	if f == nil {
		return 0
	}
	return f()
}

// RecordEnqueueFailureFunc is the equivalent of ObsMetrics.RecordEnqueueFailure.
type RecordEnqueueFailureFunc func(context.Context, int64)

var _ interface {
	RecordEnqueueFailure(context.Context, int64)
} = RecordEnqueueFailureFunc(nil)

func (f RecordEnqueueFailureFunc) RecordEnqueueFailure(ctx context.Context, items int64) {
	if f != nil {
		f(ctx, items)
	}
}

// RecordBatchSendSizeFunc is the equivalent of ObsMetrics.RecordBatchSendSize.
type RecordBatchSendSizeFunc func(context.Context, int64, int64)

var _ interface {
	RecordBatchSendSize(context.Context, int64, int64)
} = RecordBatchSendSizeFunc(nil)

func (f RecordBatchSendSizeFunc) RecordBatchSendSize(ctx context.Context, items, bytes int64) {
	if f != nil {
		f(ctx, items, bytes)
	}
}

// RegisterQueueSizeFunc is the equivalent of ObsMetrics.RegisterQueueSize.
type RegisterQueueSizeFunc func(QueueObserver) error

var _ interface {
	RegisterQueueSize(QueueObserver) error
} = RegisterQueueSizeFunc(nil)

func (f RegisterQueueSizeFunc) RegisterQueueSize(observe QueueObserver) error {
	if f == nil {
		return nil
	}
	return f(observe)
}

// RegisterQueueCapacityFunc is the equivalent of ObsMetrics.RegisterQueueCapacity.
type RegisterQueueCapacityFunc func(QueueObserver) error

var _ interface {
	RegisterQueueCapacity(QueueObserver) error
} = RegisterQueueCapacityFunc(nil)

func (f RegisterQueueCapacityFunc) RegisterQueueCapacity(observe QueueObserver) error {
	if f == nil {
		return nil
	}
	return f(observe)
}

// RecordInFlightFunc is the equivalent of ObsMetrics.RecordInFlight.
type RecordInFlightFunc func(context.Context, int64)

var _ interface {
	RecordInFlight(context.Context, int64)
} = RecordInFlightFunc(nil)

func (f RecordInFlightFunc) RecordInFlight(ctx context.Context, delta int64) {
	if f != nil {
		f(ctx, delta)
	}
}

// RecordSentFunc is the equivalent of ObsMetrics.RecordSent.
type RecordSentFunc func(context.Context, int64)

var _ interface {
	RecordSent(context.Context, int64)
} = RecordSentFunc(nil)

func (f RecordSentFunc) RecordSent(ctx context.Context, items int64) {
	if f != nil {
		f(ctx, items)
	}
}

// RecordSendFailureFunc is the equivalent of ObsMetrics.RecordSendFailure.
type RecordSendFailureFunc func(context.Context, int64, ...metric.AddOption)

var _ interface {
	RecordSendFailure(context.Context, int64, ...metric.AddOption)
} = RecordSendFailureFunc(nil)

func (f RecordSendFailureFunc) RecordSendFailure(ctx context.Context, items int64, options ...metric.AddOption) {
	if f != nil {
		f(ctx, items, options...)
	}
}

// ShutdownObsMetricsFunc is the equivalent of ObsMetrics.Shutdown.
type ShutdownObsMetricsFunc func()

var _ interface {
	Shutdown()
} = ShutdownObsMetricsFunc(nil)

func (f ShutdownObsMetricsFunc) Shutdown() {
	if f != nil {
		f()
	}
}

type obsMetrics struct {
	RecordEnqueueFailureFunc
	RecordBatchSendSizeFunc
	RegisterQueueSizeFunc
	RegisterQueueCapacityFunc
	RecordInFlightFunc
	RecordSentFunc
	RecordSendFailureFunc
	ShutdownObsMetricsFunc
}

func (*obsMetrics) unexportedObsMetricsFunc() {}

// ObsMetricsOption applies an option to ObsMetrics.
type ObsMetricsOption interface {
	apply(*obsMetrics)
}

type obsMetricsOptionFunc func(*obsMetrics)

func (f obsMetricsOptionFunc) apply(metrics *obsMetrics) {
	f(metrics)
}

// NewObsMetrics creates an ObsMetrics from operation functions. Exporterhelper
// defines the meaning and timing of each operation; options connect those
// operations to component-owned instruments. Operations without a
// corresponding option are intentionally no-ops.
func NewObsMetrics(options ...ObsMetricsOption) ObsMetrics {
	metrics := &obsMetrics{}
	for _, option := range options {
		option.apply(metrics)
	}
	return metrics
}

// WithRecordEnqueueFailure sets the function used to record enqueue failures.
func WithRecordEnqueueFailure(f RecordEnqueueFailureFunc) ObsMetricsOption {
	return obsMetricsOptionFunc(func(metrics *obsMetrics) {
		metrics.RecordEnqueueFailureFunc = f
	})
}

// WithRecordBatchSendSize sets the function used to record batch sizes.
func WithRecordBatchSendSize(f RecordBatchSendSizeFunc) ObsMetricsOption {
	return obsMetricsOptionFunc(func(metrics *obsMetrics) {
		metrics.RecordBatchSendSizeFunc = f
	})
}

// WithRegisterQueueSize sets the function used to register queue-size observation.
func WithRegisterQueueSize(f RegisterQueueSizeFunc) ObsMetricsOption {
	return obsMetricsOptionFunc(func(metrics *obsMetrics) {
		metrics.RegisterQueueSizeFunc = f
	})
}

// WithRegisterQueueCapacity sets the function used to register queue-capacity observation.
func WithRegisterQueueCapacity(f RegisterQueueCapacityFunc) ObsMetricsOption {
	return obsMetricsOptionFunc(func(metrics *obsMetrics) {
		metrics.RegisterQueueCapacityFunc = f
	})
}

// WithRecordInFlight sets the function used to record in-flight requests.
func WithRecordInFlight(f RecordInFlightFunc) ObsMetricsOption {
	return obsMetricsOptionFunc(func(metrics *obsMetrics) {
		metrics.RecordInFlightFunc = f
	})
}

// WithRecordSent sets the function used to record successfully sent items.
func WithRecordSent(f RecordSentFunc) ObsMetricsOption {
	return obsMetricsOptionFunc(func(metrics *obsMetrics) {
		metrics.RecordSentFunc = f
	})
}

// WithRecordSendFailure sets the function used to record send failures.
func WithRecordSendFailure(f RecordSendFailureFunc) ObsMetricsOption {
	return obsMetricsOptionFunc(func(metrics *obsMetrics) {
		metrics.RecordSendFailureFunc = f
	})
}

// WithObsMetricsShutdown sets the function called when ObsMetrics shuts down.
func WithObsMetricsShutdown(f ShutdownObsMetricsFunc) ObsMetricsOption {
	return obsMetricsOptionFunc(func(metrics *obsMetrics) {
		metrics.ShutdownObsMetricsFunc = f
	})
}
