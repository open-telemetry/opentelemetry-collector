// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper"

import (
	"context"

	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// NewLogsRequest creates new logs exporter based on custom LogsConverter and Sender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewLogsRequest(
	ctx context.Context,
	set exporter.Settings,
	converter RequestConverterFunc[plog.Logs],
	pusher RequestConsumeFunc,
	options ...exporterhelper.Option,
) (exporter.Logs, error) {
	return internal.NewLogsRequest(ctx, set, converter, pusher, options...)
}

// NewMetricsRequest creates new metrics exporter based on custom MetricsConverter and Sender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewMetricsRequest(
	ctx context.Context,
	set exporter.Settings,
	converter RequestConverterFunc[pmetric.Metrics],
	pusher RequestConsumeFunc,
	options ...exporterhelper.Option,
) (exporter.Metrics, error) {
	return internal.NewMetricsRequest(ctx, set, converter, pusher, options...)
}

// NewTracesRequest creates new traces exporter based on custom TracesConverter and Sender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewTracesRequest(
	ctx context.Context,
	set exporter.Settings,
	converter RequestConverterFunc[ptrace.Traces],
	pusher RequestConsumeFunc,
	options ...exporterhelper.Option,
) (exporter.Traces, error) {
	return internal.NewTracesRequest(ctx, set, converter, pusher, options...)
}

// QueueBatchSettings are settings for the QueueBatch component.
// They include things line Encoding to be used with persistent queue, or the available Sizers, etc.
type QueueBatchSettings = queuebatch.Settings[Request]

// NewMetricsQueueBatchSettings returns a new QueueBatchSettings to configure to WithQueueBatch when using pmetric.Metrics.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewMetricsQueueBatchSettings() QueueBatchSettings {
	return queuebatch.NewMetricsQueueBatchSettings()
}

// NewLogsQueueBatchSettings returns a new QueueBatchSettings to configure to WithQueueBatch when using plog.Logs.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewLogsQueueBatchSettings() QueueBatchSettings {
	return queuebatch.NewLogsQueueBatchSettings()
}

// NewTracesQueueBatchSettings returns a new QueueBatchSettings to configure to WithQueueBatch when using ptrace.Traces.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewTracesQueueBatchSettings() QueueBatchSettings {
	return queuebatch.NewTracesQueueBatchSettings()
}

// WithQueueBatch enables queueing and batching for an exporter.
// This option should be used with the new exporter helpers New[Traces|Metrics|Logs]RequestExporter.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func WithQueueBatch(cfg configoptional.Optional[exporterhelper.QueueBatchConfig], set QueueBatchSettings) exporterhelper.Option {
	return internal.WithQueueBatch(cfg, set)
}
