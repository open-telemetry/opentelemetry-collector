// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package queuebatch defines the internal contract used to report queue and
// batch telemetry.
package queuebatch // import "go.opentelemetry.io/collector/internal/telemetry/queuebatch"

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/component"
)

// Metric identifies a metric produced by queue or batch operations.
type Metric string

const (
	MetricEnqueueFailure     Metric = "enqueue_failure"
	MetricEnqueueSize        Metric = "enqueue_size"
	MetricEnqueueSizeBytes   Metric = "enqueue_size_bytes"
	MetricQueueSize          Metric = "queue_size"
	MetricQueueCapacity      Metric = "queue_capacity"
	MetricBatchSendSize      Metric = "batch_send_size"
	MetricBatchSendSizeBytes Metric = "batch_send_size_bytes"
	MetricInFlight           Metric = "in_flight"
	MetricSent               Metric = "sent"
	MetricSendFailure        Metric = "send_failure"
)

// ObsMetrics reports metrics produced by queue and batch operations.
// Nil functions disable metrics. Functions may be invoked concurrently.
type ObsMetrics struct {
	ShouldRecordFunc func(context.Context, Metric) bool
	RecordIntFunc    func(context.Context, Metric, int64, ...metric.AddOption)
	RegisterIntFunc  func(Metric, func() int64) error
	ShutdownFunc     func()
}

func (m ObsMetrics) ShouldRecord(ctx context.Context, metric Metric) bool {
	return m.ShouldRecordFunc != nil && m.ShouldRecordFunc(ctx, metric)
}

func (m ObsMetrics) RecordInt(ctx context.Context, metric Metric, value int64, options ...metric.AddOption) {
	if m.RecordIntFunc != nil {
		m.RecordIntFunc(ctx, metric, value, options...)
	}
}

func (m ObsMetrics) RegisterInt(metric Metric, value func() int64) error {
	if m.RegisterIntFunc == nil {
		return nil
	}
	return m.RegisterIntFunc(metric, value)
}

func (m ObsMetrics) Shutdown() {
	if m.ShutdownFunc != nil {
		m.ShutdownFunc()
	}
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
