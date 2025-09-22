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
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// NewTracesQueueBatchSettings returns a new QueueBatchSettings to configure to WithQueueBatch when using ptrace.Traces.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
// Deprecated: [v0.136.0] Use xexporterhelper.NewTracesQueueBatchSettings instead.
func NewTracesQueueBatchSettings() QueueBatchSettings {
	return queuebatch.NewTracesQueueBatchSettings()
}

// NewTraces creates an exporter.Traces that records observability metrics and wraps every request with a Span.
func NewTraces(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
	pusher consumer.ConsumeTracesFunc,
	options ...Option,
) (exporter.Traces, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	if pusher == nil {
		return nil, errNilPushTraces
	}
	return internal.NewTracesRequest(ctx, set, queuebatch.RequestFromTraces(), queuebatch.RequestConsumeFromTraces(pusher),
		append([]Option{internal.WithQueueBatchSettings(NewTracesQueueBatchSettings())}, options...)...)
}

// NewTracesRequest creates a new traces exporter based on a custom TracesConverter and Sender.
// Deprecated [v0.136.0]: Use xexporterhelper.NewTracesRequest instead.
func NewTracesRequest(
	ctx context.Context,
	set exporter.Settings,
	converter RequestConverterFunc[ptrace.Traces],
	pusher RequestConsumeFunc,
	options ...Option,
) (exporter.Traces, error) {
	return internal.NewTracesRequest(ctx, set, converter, pusher, options...)
}
