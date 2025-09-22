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
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// NewMetricsQueueBatchSettings returns a new QueueBatchSettings to configure to WithQueueBatch when using pmetric.Metrics.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
// Deprecated [v0.136.0]: Use xexporterhelper.NewMetricsQueueBatchSettings instead.
func NewMetricsQueueBatchSettings() QueueBatchSettings {
	return queuebatch.NewMetricsQueueBatchSettings()
}

// NewMetrics creates an exporter.Metrics that records observability metrics and wraps every request with a Span.
func NewMetrics(
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
		append([]Option{internal.WithQueueBatchSettings(NewMetricsQueueBatchSettings())}, options...)...)
}

// NewMetricsRequest creates a new metrics exporter based on a custom MetricsConverter and Sender.
// Deprecated [v0.136.0]: Use xexporterhelper.NewMetricsRequest instead.
func NewMetricsRequest(
	ctx context.Context,
	set exporter.Settings,
	converter RequestConverterFunc[pmetric.Metrics],
	pusher RequestConsumeFunc,
	options ...Option,
) (exporter.Metrics, error) {
	return internal.NewMetricsRequest(ctx, set, converter, pusher, options...)
}
