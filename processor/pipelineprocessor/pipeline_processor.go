// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pipelineprocessor // import "go.opentelemetry.io/collector/processor/pipelineprocessor"

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
)

// newTracesProcessor creates a new traces processor that uses exporterhelper capabilities.
func newTracesProcessor(set processor.Settings, nextConsumer consumer.Traces, cfg *Config) (processor.Traces, error) {
	// Create a push function that forwards to the next consumer
	pushFunc := func(ctx context.Context, td ptrace.Traces) error {
		return nextConsumer.ConsumeTraces(ctx, td)
	}

	// Convert processor settings to exporter settings
	exporterSet := exporter.Settings{
		ID:                set.ID,
		TelemetrySettings: set.TelemetrySettings,
		BuildInfo:         set.BuildInfo,
	}

	// Use exporterhelper to create a traces exporter with all the features
	return exporterhelper.NewTraces(
		context.Background(),
		exporterSet,
		cfg,
		pushFunc,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(cfg.TimeoutConfig),
		exporterhelper.WithRetry(cfg.RetryConfig),
		exporterhelper.WithQueue(cfg.QueueConfig),
	)
}

// newMetricsProcessor creates a new metrics processor that uses exporterhelper capabilities.
func newMetricsProcessor(set processor.Settings, nextConsumer consumer.Metrics, cfg *Config) (processor.Metrics, error) {
	// Create a push function that forwards to the next consumer
	pushFunc := func(ctx context.Context, md pmetric.Metrics) error {
		return nextConsumer.ConsumeMetrics(ctx, md)
	}

	// Convert processor settings to exporter settings
	exporterSet := exporter.Settings{
		ID:                set.ID,
		TelemetrySettings: set.TelemetrySettings,
		BuildInfo:         set.BuildInfo,
	}

	// Use exporterhelper to create a metrics exporter with all the features
	return exporterhelper.NewMetrics(
		context.Background(),
		exporterSet,
		cfg,
		pushFunc,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(cfg.TimeoutConfig),
		exporterhelper.WithRetry(cfg.RetryConfig),
		exporterhelper.WithQueue(cfg.QueueConfig),
	)
}

// newLogsProcessor creates a new logs processor that uses exporterhelper capabilities.
func newLogsProcessor(set processor.Settings, nextConsumer consumer.Logs, cfg *Config) (processor.Logs, error) {
	// Create a push function that forwards to the next consumer
	pushFunc := func(ctx context.Context, ld plog.Logs) error {
		return nextConsumer.ConsumeLogs(ctx, ld)
	}

	// Convert processor settings to exporter settings
	exporterSet := exporter.Settings{
		ID:                set.ID,
		TelemetrySettings: set.TelemetrySettings,
		BuildInfo:         set.BuildInfo,
	}

	// Use exporterhelper to create a logs exporter with all the features
	return exporterhelper.NewLogs(
		context.Background(),
		exporterSet,
		cfg,
		pushFunc,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(cfg.TimeoutConfig),
		exporterhelper.WithRetry(cfg.RetryConfig),
		exporterhelper.WithQueue(cfg.QueueConfig),
	)
}
