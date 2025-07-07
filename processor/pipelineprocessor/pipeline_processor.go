// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pipelineprocessor // import "go.opentelemetry.io/collector/processor/pipelineprocessor"

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/processor"
)

// newTracesProcessor creates a new traces processor that uses exporterhelper capabilities.
func newTracesProcessor(set processor.Settings, nextConsumer consumer.Traces, cfg *Config) (processor.Traces, error) {
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
		nextConsumer.ConsumeTraces,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(cfg.TimeoutConfig),
		exporterhelper.WithRetry(cfg.RetryConfig),
		exporterhelper.WithQueueBatch(cfg.QueueConfig, exporterhelper.NewTracesQueueBatchSettings()),
	)
}

// newMetricsProcessor creates a new metrics processor that uses exporterhelper capabilities.
func newMetricsProcessor(set processor.Settings, nextConsumer consumer.Metrics, cfg *Config) (processor.Metrics, error) {
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
		nextConsumer.ConsumeMetrics,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(cfg.TimeoutConfig),
		exporterhelper.WithRetry(cfg.RetryConfig),
		exporterhelper.WithQueueBatch(cfg.QueueConfig, exporterhelper.NewMetricsQueueBatchSettings()),
	)
}

// newLogsProcessor creates a new logs processor that uses exporterhelper capabilities.
func newLogsProcessor(set processor.Settings, nextConsumer consumer.Logs, cfg *Config) (processor.Logs, error) {
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
		nextConsumer.ConsumeLogs,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(cfg.TimeoutConfig),
		exporterhelper.WithRetry(cfg.RetryConfig),
		exporterhelper.WithQueueBatch(cfg.QueueConfig, exporterhelper.NewLogsQueueBatchSettings()),
	)
}
