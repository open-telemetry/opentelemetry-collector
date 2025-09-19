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
	"go.opentelemetry.io/collector/pdata/plog"
)

// NewLogsQueueBatchSettings returns a new QueueBatchSettings to configure to WithQueueBatch when using plog.Logs.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
// Deprecated: [v0.136.0] Use xexporterhelper.NewLogsQueueBatchSettings instead.
func NewLogsQueueBatchSettings() QueueBatchSettings {
	return queuebatch.NewLogsQueueBatchSettings()
}

// NewLogs creates an exporter.Logs that records observability logs and wraps every request with a Span.
func NewLogs(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
	pusher consumer.ConsumeLogsFunc,
	options ...Option,
) (exporter.Logs, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	if pusher == nil {
		return nil, errNilPushLogs
	}
	return internal.NewLogsRequest(ctx, set, queuebatch.RequestFromLogs(), queuebatch.RequestConsumeFromLogs(pusher),
		append([]Option{internal.WithQueueBatchSettings(NewLogsQueueBatchSettings())}, options...)...)
}

// NewLogsRequest creates new logs exporter based on custom LogsConverter and Sender.
// Deprecated [v0.136.0]: Use xexporterhelper.NewLogsRequest instead.
func NewLogsRequest(
	ctx context.Context,
	set exporter.Settings,
	converter RequestConverterFunc[plog.Logs],
	pusher RequestConsumeFunc,
	options ...Option,
) (exporter.Logs, error) {
	return internal.NewLogsRequest(ctx, set, converter, pusher, options...)
}
