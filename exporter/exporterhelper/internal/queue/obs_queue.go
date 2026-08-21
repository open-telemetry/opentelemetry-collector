// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

// QueueMetrics reports the metrics produced by a Queue; use NewQueueMetrics.
type QueueMetrics interface {
	// RecordEnqueueFailure counts items the queue rejected.
	RecordEnqueueFailure(ctx context.Context, items int64)
	// RecordEnqueueSize counts the items and bytes offered to the queue.
	RecordEnqueueSize(ctx context.Context, items int64, bytesSize Int64Value)
	// RegisterQueueSize installs an observer for the current queue size.
	RegisterQueueSize(observeSize Int64Value) error
	// RegisterQueueCapacity installs an observer for the fixed queue capacity.
	RegisterQueueCapacity(observeCapacity Int64Value) error
}

// Int64Value supplies an int64 value; use NewInt64Value.
type Int64Value interface {
	Value() int64

	private()
}

// ValueFunc supplies an int64 value.
type ValueFunc func() int64

func (f ValueFunc) Value() int64 {
	if f == nil {
		return 0
	}
	return f()
}

func (ValueFunc) private() {}

// NewInt64Value returns an Int64Value backed by value.
func NewInt64Value(value ValueFunc) Int64Value {
	return value
}

// RecordEnqueueFailureFunc records the items dropped because the queue rejected them.
type RecordEnqueueFailureFunc func(ctx context.Context, items int64)

func (f RecordEnqueueFailureFunc) RecordEnqueueFailure(ctx context.Context, items int64) {
	if f == nil {
		return
	}
	f(ctx, items)
}

// RecordEnqueueSizeFunc records a request offered to the queue, calling bytesSize only if reported.
type RecordEnqueueSizeFunc func(ctx context.Context, items int64, bytesSize Int64Value)

func (f RecordEnqueueSizeFunc) RecordEnqueueSize(ctx context.Context, items int64, bytesSize Int64Value) {
	if f == nil {
		return
	}
	f(ctx, items, bytesSize)
}

// RegisterQueueSizeFunc installs an observer for the current queue size.
type RegisterQueueSizeFunc func(observeSize Int64Value) error

func (f RegisterQueueSizeFunc) RegisterQueueSize(observeSize Int64Value) error {
	if f == nil {
		return nil
	}
	return f(observeSize)
}

// RegisterQueueCapacityFunc installs an observer for the fixed queue capacity.
type RegisterQueueCapacityFunc func(observeCapacity Int64Value) error

func (f RegisterQueueCapacityFunc) RegisterQueueCapacity(observeCapacity Int64Value) error {
	if f == nil {
		return nil
	}
	return f(observeCapacity)
}

// QueueMetricsOption configures QueueMetrics.
type QueueMetricsOption interface {
	applyQueueMetrics(*queueMetrics)
}

type queueMetricsOptionFunc func(*queueMetrics)

func (f queueMetricsOptionFunc) applyQueueMetrics(metrics *queueMetrics) {
	f(metrics)
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

// queueMetrics implements QueueMetrics from a set of operations.
type queueMetrics struct {
	RecordEnqueueFailureFunc
	RecordEnqueueSizeFunc
	RegisterQueueSizeFunc
	RegisterQueueCapacityFunc
}

var _ QueueMetrics = queueMetrics{}

// obsQueue is a helper to add observability to a queue.
type obsQueue[T request.Request] struct {
	Queue[T]
	queueMetrics QueueMetrics
	tracer       trace.Tracer
}

func newObsQueue[T request.Request](set Settings[T], delegate Queue[T]) (Queue[T], error) {
	qm := set.QueueMetrics
	if qm == nil {
		qm = NewQueueMetrics()
	}

	if err := qm.RegisterQueueSize(NewInt64Value(delegate.Size)); err != nil {
		return nil, err
	}

	if err := qm.RegisterQueueCapacity(NewInt64Value(delegate.Capacity)); err != nil {
		return nil, err
	}

	return &obsQueue[T]{
		Queue:        delegate,
		queueMetrics: qm,
		tracer:       metadata.Tracer(set.Telemetry),
	}, nil
}

func (or *obsQueue[T]) Offer(ctx context.Context, req T) error {
	// Have to read the number of items before sending the request since the request can
	// be modified by the downstream components like the batcher.
	numItems := req.ItemsCount()

	or.queueMetrics.RecordEnqueueSize(ctx, int64(numItems), NewInt64Value(func() int64 {
		return int64(req.BytesSize())
	}))

	ctx, span := or.tracer.Start(ctx, "exporter/enqueue")
	err := or.Queue.Offer(ctx, req)
	span.End()

	if err != nil {
		or.queueMetrics.RecordEnqueueFailure(ctx, int64(numItems))
	}
	return err
}
