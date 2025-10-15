// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"context"
	"runtime"

	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/processor"
)

// translateToExporterHelperConfig converts legacy batchprocessor config to exporterhelper config
func translateToExporterHelperConfig(cfg *Config) exporterhelper.QueueBatchConfig {
	// These settings match legacy behavior
	queueBatchConfig := exporterhelper.QueueBatchConfig{
		Enabled:         true,
		WaitForResult:   propagateErrors.IsEnabled(),
		BlockOnOverflow: true,
		Sizer:           exporterhelper.RequestSizerTypeItems,
		QueueSize:       int64(runtime.NumCPU()) * int64(max(cfg.SendBatchSize, cfg.SendBatchMaxSize, 100)),
		// Note: users could request to raise this, we could support a new
		// option to add concurrency.
		NumConsumers:    1,
	}

	if cfg.SendBatchSize > 0 || cfg.SendBatchMaxSize > 0 || cfg.Timeout > 0 {
		batchConfig := exporterhelper.BatchConfig{
			FlushTimeout: cfg.Timeout,
			Sizer:        exporterhelper.RequestSizerTypeItems,
			MinSize:      0, // Default: no minimum
			MaxSize:      0, // Default: no maximum
		}

		// Map send_batch_size to MinSize (minimum items to trigger batch)
		if cfg.SendBatchSize > 0 {
			batchConfig.MinSize = int64(cfg.SendBatchSize)
		}

		// Map send_batch_max_size to MaxSize (maximum items in batch)
		if cfg.SendBatchMaxSize > 0 {
			batchConfig.MaxSize = int64(cfg.SendBatchMaxSize)
		}

		queueBatchConfig.Batch = configoptional.Some(batchConfig)
	}

	return queueBatchConfig
}

// newTracesProcessorWithExporterHelper creates a new traces processor using exporterhelper components.
func newTracesProcessorWithExporterHelper(ctx context.Context, set processor.Settings, nextConsumer consumer.Traces, cfg *Config) (processor.Traces, error) {
	set.Logger.Info("Creating traces processor with ExporterHelper")

	queueBatchConfig := translateToExporterHelperConfig(cfg)

	exporterSet := exporter.Settings{
		ID:                set.ID,
		TelemetrySettings: set.TelemetrySettings,
		BuildInfo:         set.BuildInfo,
	}

	result, err := exporterhelper.NewTraces(
		ctx,
		exporterSet,
		cfg,
		nextConsumer.ConsumeTraces,
		exporterhelper.WithQueue(queueBatchConfig),
	)
	return result, err
}

// newMetricsProcessorWithExporterHelper creates a new metrics processor using exporterhelper components.
func newMetricsProcessorWithExporterHelper(ctx context.Context, set processor.Settings, nextConsumer consumer.Metrics, cfg *Config) (processor.Metrics, error) {
	queueBatchConfig := translateToExporterHelperConfig(cfg)

	exporterSet := exporter.Settings{
		ID:                set.ID,
		TelemetrySettings: set.TelemetrySettings,
		BuildInfo:         set.BuildInfo,
	}

	return exporterhelper.NewMetrics(
		ctx,
		exporterSet,
		cfg,
		nextConsumer.ConsumeMetrics,
		exporterhelper.WithQueue(queueBatchConfig),
	)
}

// newLogsProcessorWithExporterHelper creates a new logs processor using exporterhelper components.
func newLogsProcessorWithExporterHelper(ctx context.Context, set processor.Settings, nextConsumer consumer.Logs, cfg *Config) (processor.Logs, error) {
	queueBatchConfig := translateToExporterHelperConfig(cfg)

	exporterSet := exporter.Settings{
		ID:                set.ID,
		TelemetrySettings: set.TelemetrySettings,
		BuildInfo:         set.BuildInfo,
	}

	return exporterhelper.NewLogs(
		ctx,
		exporterSet,
		cfg,
		nextConsumer.ConsumeLogs,
		exporterhelper.WithQueue(queueBatchConfig),
	)
}
