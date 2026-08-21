// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
)

// ObsMetrics reports the metrics produced by exporterhelper for one signal; use NewObsMetrics.
type ObsMetrics interface {
	QueueBatchMetrics

	RecordInFlight(ctx context.Context, delta int64)
	RecordSent(ctx context.Context, items int64)
	RecordSendFailure(ctx context.Context, items int64, options ...metric.AddOption)
	Shutdown()
}

// QueueMetrics reports the metrics produced by the sending queue; use NewQueueMetrics.
type QueueMetrics interface {
	RecordEnqueueFailure(ctx context.Context, items int64)
	RecordEnqueueSize(ctx context.Context, items int64, bytesSize Int64Value)
	RegisterQueueSize(observeSize Int64Value) error
	RegisterQueueCapacity(observeCapacity Int64Value) error

	private()
}

// QueueBatchMetrics reports the queue and batcher metrics; use NewQueueBatchMetrics.
type QueueBatchMetrics interface {
	QueueMetrics

	RecordBatchSendSize(ctx context.Context, items int64, bytesSize Int64Value)
}

// Int64Value supplies an int64 value; use NewInt64Value.
type Int64Value interface {
	Value() int64

	private()
}

// ValueFunc supplies an int64 value. A nil ValueFunc returns zero.
type ValueFunc func() int64

// Value returns the supplied value, or zero if f is nil.
func (f ValueFunc) Value() int64 {
	if f == nil {
		return 0
	}
	return f()
}

func (ValueFunc) private() {}

// RecordEnqueueFailureFunc records the items dropped because the queue rejected them.
// A nil RecordEnqueueFailureFunc reports nothing.
type RecordEnqueueFailureFunc func(ctx context.Context, items int64)

// RecordEnqueueFailure records an enqueue failure if f is non-nil.
func (f RecordEnqueueFailureFunc) RecordEnqueueFailure(ctx context.Context, items int64) {
	if f != nil {
		f(ctx, items)
	}
}

// RecordEnqueueSizeFunc records a request offered to the queue, calling bytesSize only if reported.
// A nil RecordEnqueueSizeFunc reports nothing.
type RecordEnqueueSizeFunc func(ctx context.Context, items int64, bytesSize Int64Value)

// RecordEnqueueSize records the enqueue size if f is non-nil.
func (f RecordEnqueueSizeFunc) RecordEnqueueSize(ctx context.Context, items int64, bytesSize Int64Value) {
	if f != nil {
		f(ctx, items, bytesSize)
	}
}

// RegisterQueueSizeFunc installs an observer for the current queue size.
// A nil RegisterQueueSizeFunc returns nil without installing an observer.
type RegisterQueueSizeFunc func(observeSize Int64Value) error

// RegisterQueueSize registers the queue-size observer if f is non-nil.
func (f RegisterQueueSizeFunc) RegisterQueueSize(observeSize Int64Value) error {
	if f == nil {
		return nil
	}
	return f(observeSize)
}

// RegisterQueueCapacityFunc installs an observer for the fixed queue capacity.
// A nil RegisterQueueCapacityFunc returns nil without installing an observer.
type RegisterQueueCapacityFunc func(observeCapacity Int64Value) error

// RegisterQueueCapacity registers the queue-capacity observer if f is non-nil.
func (f RegisterQueueCapacityFunc) RegisterQueueCapacity(observeCapacity Int64Value) error {
	if f == nil {
		return nil
	}
	return f(observeCapacity)
}

// RecordBatchSendSizeFunc records a batch as it is sent, calling bytesSize only if reported.
// A nil RecordBatchSendSizeFunc reports nothing.
type RecordBatchSendSizeFunc func(ctx context.Context, items int64, bytesSize Int64Value)

// RecordBatchSendSize records the batch send size if f is non-nil.
func (f RecordBatchSendSizeFunc) RecordBatchSendSize(ctx context.Context, items int64, bytesSize Int64Value) {
	if f != nil {
		f(ctx, items, bytesSize)
	}
}

// RecordInFlightFunc records a change in the requests being sent, +1 at start and -1 at end.
// A nil RecordInFlightFunc reports nothing.
type RecordInFlightFunc func(ctx context.Context, delta int64)

// RecordInFlight records the in-flight change if f is non-nil.
func (f RecordInFlightFunc) RecordInFlight(ctx context.Context, delta int64) {
	if f != nil {
		f(ctx, delta)
	}
}

// RecordSentFunc records the number of items successfully sent.
// A nil RecordSentFunc reports nothing.
type RecordSentFunc func(ctx context.Context, items int64)

// RecordSent records the sent items if f is non-nil.
func (f RecordSentFunc) RecordSent(ctx context.Context, items int64) {
	if f != nil {
		f(ctx, items)
	}
}

// RecordSendFailureFunc records the items that failed to send, with the failure attributes.
// A nil RecordSendFailureFunc reports nothing.
type RecordSendFailureFunc func(ctx context.Context, items int64, options ...metric.AddOption)

// RecordSendFailure records the send failure if f is non-nil.
func (f RecordSendFailureFunc) RecordSendFailure(ctx context.Context, items int64, options ...metric.AddOption) {
	if f != nil {
		f(ctx, items, options...)
	}
}

// ShutdownFunc releases the resources backing the instruments.
// A nil ShutdownFunc releases nothing.
type ShutdownFunc func()

// Shutdown releases metric resources if f is non-nil.
func (f ShutdownFunc) Shutdown() {
	if f != nil {
		f()
	}
}

// QueueMetricsOption configures QueueMetrics.
type QueueMetricsOption interface {
	applyQueueMetrics(*queueMetrics)
}

type queueMetricsOptionFunc func(*queueMetrics)

func (f queueMetricsOptionFunc) applyQueueMetrics(metrics *queueMetrics) {
	f(metrics)
}

// QueueBatchMetricsOption configures QueueBatchMetrics.
type QueueBatchMetricsOption interface {
	applyQueueBatchMetrics(*queueBatchMetrics)
}

type queueBatchMetricsOptionFunc func(*queueBatchMetrics)

func (f queueBatchMetricsOptionFunc) applyQueueBatchMetrics(metrics *queueBatchMetrics) {
	f(metrics)
}

// ObsMetricsOption configures ObsMetrics.
type ObsMetricsOption interface {
	applyObsMetrics(*obsMetrics)
}

type obsMetricsOptionFunc func(*obsMetrics)

func (f obsMetricsOptionFunc) applyObsMetrics(metrics *obsMetrics) {
	f(metrics)
}

// NewInt64Value returns an Int64Value backed by value.
func NewInt64Value(value ValueFunc) Int64Value {
	return value
}

// WithRecordEnqueueFailure configures how enqueue failures are recorded.
func WithRecordEnqueueFailure(record RecordEnqueueFailureFunc) QueueMetricsOption {
	return queueMetricsOptionFunc(func(metrics *queueMetrics) {
		metrics.RecordEnqueueFailureFunc = record
	})
}

// WithRecordEnqueueSize configures how enqueue sizes are recorded.
func WithRecordEnqueueSize(record RecordEnqueueSizeFunc) QueueMetricsOption {
	return queueMetricsOptionFunc(func(metrics *queueMetrics) {
		metrics.RecordEnqueueSizeFunc = record
	})
}

// WithRegisterQueueSize configures how the queue-size observer is registered.
func WithRegisterQueueSize(register RegisterQueueSizeFunc) QueueMetricsOption {
	return queueMetricsOptionFunc(func(metrics *queueMetrics) {
		metrics.RegisterQueueSizeFunc = register
	})
}

// WithRegisterQueueCapacity configures how the queue-capacity observer is registered.
func WithRegisterQueueCapacity(register RegisterQueueCapacityFunc) QueueMetricsOption {
	return queueMetricsOptionFunc(func(metrics *queueMetrics) {
		metrics.RegisterQueueCapacityFunc = register
	})
}

// NewQueueMetrics returns QueueMetrics whose unspecified operations report nothing.
func NewQueueMetrics(options ...QueueMetricsOption) QueueMetrics {
	metrics := queueMetrics{}
	for _, option := range options {
		option.applyQueueMetrics(&metrics)
	}
	return metrics
}

// WithQueueMetrics configures the queue-level metrics.
func WithQueueMetrics(metrics QueueMetrics) QueueBatchMetricsOption {
	return queueBatchMetricsOptionFunc(func(batchMetrics *queueBatchMetrics) {
		if metrics != nil {
			batchMetrics.QueueMetrics = metrics
		}
	})
}

// WithRecordBatchSendSize configures how batch send sizes are recorded.
func WithRecordBatchSendSize(record RecordBatchSendSizeFunc) QueueBatchMetricsOption {
	return queueBatchMetricsOptionFunc(func(metrics *queueBatchMetrics) {
		metrics.RecordBatchSendSizeFunc = record
	})
}

// NewQueueBatchMetrics returns QueueBatchMetrics whose unspecified operations report nothing.
func NewQueueBatchMetrics(options ...QueueBatchMetricsOption) QueueBatchMetrics {
	metrics := queueBatchMetrics{QueueMetrics: NewQueueMetrics()}
	for _, option := range options {
		option.applyQueueBatchMetrics(&metrics)
	}
	return metrics
}

// WithQueueBatchMetrics configures the queue and batch metrics.
func WithQueueBatchMetrics(metrics QueueBatchMetrics) ObsMetricsOption {
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
	metrics := obsMetrics{QueueBatchMetrics: NewQueueBatchMetrics()}
	for _, option := range options {
		option.applyObsMetrics(&metrics)
	}
	return metrics
}

type queueMetrics struct {
	RecordEnqueueFailureFunc
	RecordEnqueueSizeFunc
	RegisterQueueSizeFunc
	RegisterQueueCapacityFunc
}

func (queueMetrics) private() {}

type queueBatchMetrics struct {
	QueueMetrics
	RecordBatchSendSizeFunc
}

type obsMetrics struct {
	QueueBatchMetrics
	RecordInFlightFunc
	RecordSentFunc
	RecordSendFailureFunc
	ShutdownFunc
}

type int64ValueAdapter struct {
	queue.Int64Value
}

func (int64ValueAdapter) private() {}

func adaptInt64Value(value queue.Int64Value) Int64Value {
	if value == nil {
		return nil
	}
	return int64ValueAdapter{Int64Value: value}
}

type obsMetricsAdapter struct {
	ObsMetrics
}

func (a obsMetricsAdapter) RecordEnqueueSize(ctx context.Context, items int64, bytesSize queue.Int64Value) {
	a.ObsMetrics.RecordEnqueueSize(ctx, items, adaptInt64Value(bytesSize))
}

func (a obsMetricsAdapter) RegisterQueueSize(observeSize queue.Int64Value) error {
	return a.ObsMetrics.RegisterQueueSize(adaptInt64Value(observeSize))
}

func (a obsMetricsAdapter) RegisterQueueCapacity(observeCapacity queue.Int64Value) error {
	return a.ObsMetrics.RegisterQueueCapacity(adaptInt64Value(observeCapacity))
}

func (a obsMetricsAdapter) RecordBatchSendSize(ctx context.Context, items int64, bytesSize queue.Int64Value) {
	a.ObsMetrics.RecordBatchSendSize(ctx, items, adaptInt64Value(bytesSize))
}

func adaptObsMetrics(metrics ObsMetrics) internal.ObsMetrics {
	if metrics == nil {
		return nil
	}
	return obsMetricsAdapter{ObsMetrics: metrics}
}

var (
	_ ObsMetrics          = obsMetrics{}
	_ internal.ObsMetrics = obsMetricsAdapter{}
)
