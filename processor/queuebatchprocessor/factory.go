// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate mdatagen metadata.yaml

package queuebatchprocessor // import "go.opentelemetry.io/collector/processor/queuebatchprocessor"

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/queuebatchprocessor/internal/metadata"
)

const (
	defaultFlushTimeout = 200 * time.Millisecond
	defaultMinSize      = 8192
	defaultQueueSize    = 1000
	defaultNumConsumers = 1
)

// NewFactory returns a new factory for the queue/batch processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		processor.WithTraces(createTraces, metadata.TracesStability),
		processor.WithMetrics(createMetrics, metadata.MetricsStability),
		processor.WithLogs(createLogs, metadata.LogsStability),
	)
}

// createDefaultConfig returns the default configuration. The defaults preserve
// the batchprocessor's blocking behavior: errors propagate back to the caller
// (wait_for_result) and the processor applies backpressure when full
// (block_on_overflow), with batching enabled.
func createDefaultConfig() component.Config {
	return &Config{
		WaitForResult:   true,
		BlockOnOverflow: true,
		Sizer:           exporterhelper.RequestSizerTypeRequests,
		QueueSize:       defaultQueueSize,
		NumConsumers:    defaultNumConsumers,
		Batch: configoptional.Some(exporterhelper.BatchConfig{
			FlushTimeout: defaultFlushTimeout,
			Sizer:        exporterhelper.RequestSizerTypeItems,
			MinSize:      defaultMinSize,
		}),
	}
}

func createTraces(ctx context.Context, set processor.Settings, cfg component.Config, next consumer.Traces) (processor.Traces, error) {
	return newTracesProcessor(ctx, set, cfg.(*Config), next)
}

func createMetrics(ctx context.Context, set processor.Settings, cfg component.Config, next consumer.Metrics) (processor.Metrics, error) {
	return newMetricsProcessor(ctx, set, cfg.(*Config), next)
}

func createLogs(ctx context.Context, set processor.Settings, cfg component.Config, next consumer.Logs) (processor.Logs, error) {
	return newLogsProcessor(ctx, set, cfg.(*Config), next)
}
