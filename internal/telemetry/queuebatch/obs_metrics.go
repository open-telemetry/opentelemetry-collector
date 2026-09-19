// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package queuebatch defines the internal callbacks used to report queue and
// batch telemetry.
//
// This package intentionally uses concrete function fields instead of the
// public interface patterns described in docs/rfcs/component-interfaces.md.
// Repository-internal visibility allows this simpler contract. If external
// implementations are needed, a public interface should be designed according
// to those guidelines instead of exposing these callbacks.
package queuebatch // import "go.opentelemetry.io/collector/internal/telemetry/queuebatch"

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/component"
)

type EnqueueFailureFunc func(context.Context, int64)

func (f EnqueueFailureFunc) EnqueueFailure(ctx context.Context, items int64) {
	if f != nil {
		f(ctx, items)
	}
}

type EnqueueSizeFunc func(context.Context, int64, func() int64)

func (f EnqueueSizeFunc) EnqueueSize(ctx context.Context, items int64, bytesSize func() int64) {
	if f != nil {
		f(ctx, items, bytesSize)
	}
}

type RegisterQueueFunc func(size, capacity func() int64) error

func (f RegisterQueueFunc) RegisterQueue(size, capacity func() int64) error {
	if f == nil {
		return nil
	}
	return f(size, capacity)
}

type BatchSendSizeFunc func(context.Context, int64, func() int64)

func (f BatchSendSizeFunc) BatchSendSize(ctx context.Context, items int64, bytesSize func() int64) {
	if f != nil {
		f(ctx, items, bytesSize)
	}
}

type InFlightFunc func(context.Context, int64)

func (f InFlightFunc) InFlight(ctx context.Context, delta int64) {
	if f != nil {
		f(ctx, delta)
	}
}

type SentFunc func(context.Context, int64)

func (f SentFunc) Sent(ctx context.Context, items int64) {
	if f != nil {
		f(ctx, items)
	}
}

type SendFailureFunc func(context.Context, int64, ...metric.AddOption)

func (f SendFailureFunc) SendFailure(ctx context.Context, items int64, options ...metric.AddOption) {
	if f != nil {
		f(ctx, items, options...)
	}
}

type ShutdownFunc func()

func (f ShutdownFunc) Shutdown() {
	if f != nil {
		f()
	}
}

// QueueMetrics reports metrics produced by queue operations.
type QueueMetrics struct {
	EnqueueFailureFunc
	EnqueueSizeFunc
	RegisterQueueFunc
	ShutdownFunc
}

// SendMetrics reports metrics produced by batch and send operations.
type SendMetrics struct {
	BatchSendSizeFunc
	InFlightFunc
	SentFunc
	SendFailureFunc
}

// ObsMetrics reports the metrics produced by queue/batch operations.
// Nil callbacks disable the corresponding metrics. Callbacks may be invoked
// concurrently. Shutdown must be safe to call more than once and must release
// registrations even when RegisterQueue returns an error.
type ObsMetrics struct {
	QueueMetrics
	SendMetrics
}

type obsMetricsConfig struct {
	config     component.Config
	obsMetrics ObsMetrics
}

// ConfigWithObsMetrics attaches metrics to cfg for exporterhelper.
func ConfigWithObsMetrics(cfg component.Config, obsMetrics ObsMetrics) component.Config {
	if cfg == nil {
		return nil
	}
	return obsMetricsConfig{config: cfg, obsMetrics: obsMetrics}
}

// ObsMetricsFromConfig removes and returns metrics attached by ConfigWithObsMetrics.
func ObsMetricsFromConfig(cfg component.Config) (component.Config, ObsMetrics, bool) {
	wrapped, ok := cfg.(obsMetricsConfig)
	if !ok {
		return cfg, ObsMetrics{}, false
	}
	return wrapped.config, wrapped.obsMetrics, true
}
