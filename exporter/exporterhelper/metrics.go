// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
)

// NewMetrics creates an exporter.Metrics that records observability metrics and wraps every request with a Span.
//
// Deprecated: [v0.159.0] Use NewMetricsWithConfig instead. In a future release NewMetrics will change to drop the cfg parameter.
//
//go:fix inline
func NewMetrics(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
	pusher consumer.ConsumeMetricsFunc,
	options ...Option,
) (exporter.Metrics, error) {
	return NewMetricsWithConfig(ctx, set, cfg, pusher, options...)
}

// NewMetricsWithConfig creates an exporter.Metrics that records observability metrics and wraps every request with a Span.
func NewMetricsWithConfig(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
	pusher consumer.ConsumeMetricsFunc,
	options ...Option,
) (exporter.Metrics, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	if pusher == nil {
		return nil, errNilPushMetrics
	}
	return internal.NewMetricsRequest(ctx, set, queuebatch.RequestFromMetrics(), queuebatch.RequestConsumeFromMetrics(pusher),
		append([]Option{internal.WithQueueBatchSettings(queuebatch.NewMetricsQueueBatchSettings())}, options...)...)
}
