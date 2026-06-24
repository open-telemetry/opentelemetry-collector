// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate mdatagen metadata.yaml

package queuebatchprocessor // import "go.opentelemetry.io/collector/processor/queuebatchprocessor"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/queuebatchprocessor/internal/metadata"
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

// createDefaultConfig returns the default configuration, modified from
// exporterhelper.NewDefaultQueueConfig() to enable backpressure, error
// propagation, and limit concurrency to 1. Batching is also enabled by
// default, since that is the purpose of this component; NewDefaultQueueConfig
// leaves it as a not-yet-enabled default (pending the exporterhelper
// batching-by-default feature gate).
func createDefaultConfig() component.Config {
	cfg := exporterhelper.NewDefaultQueueConfig()
	cfg.WaitForResult = true
	cfg.BlockOnOverflow = true
	cfg.NumConsumers = 1
	cfg.Batch.GetOrInsertDefault()
	return &cfg
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
