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
		append([]Option{internal.WithQueueBatchSettings(queuebatch.NewLogsQueueBatchSettings())}, options...)...)
}
