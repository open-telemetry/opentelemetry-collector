// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"context"

	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
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
		QueueSize:       int64(max(cfg.SendBatchSize, cfg.SendBatchMaxSize, 1000)),
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

// createProcessorTelemetrySettings creates a TelemetrySettings that applies metric views
// to suppress all ExporterHelper metrics since we'll use the original batch processor telemetry.
func createProcessorTelemetrySettings(original component.TelemetrySettings) component.TelemetrySettings {
	// Define the exact scope used by ExporterHelper
	exporterHelperScope := instrumentation.Scope{
		Name: "go.opentelemetry.io/collector/exporter/exporterhelper",
	}

	// Create metric views to suppress all ExporterHelper metrics
	// We'll use the original batch processor telemetry instead for consistency
	views := []metric.View{
		// Suppress all ExporterHelper metrics by dropping them
		metric.NewView(
			metric.Instrument{
				Name:  "otelcol_exporter_enqueue_failed_log_records",
				Scope: exporterHelperScope,
			},
			metric.Stream{Aggregation: metric.AggregationDrop{}},
		),
		metric.NewView(
			metric.Instrument{
				Name:  "otelcol_exporter_enqueue_failed_metric_points",
				Scope: exporterHelperScope,
			},
			metric.Stream{Aggregation: metric.AggregationDrop{}},
		),
		metric.NewView(
			metric.Instrument{
				Name:  "otelcol_exporter_enqueue_failed_spans",
				Scope: exporterHelperScope,
			},
			metric.Stream{Aggregation: metric.AggregationDrop{}},
		),
		metric.NewView(
			metric.Instrument{
				Name:  "otelcol_exporter_send_failed_log_records",
				Scope: exporterHelperScope,
			},
			metric.Stream{Aggregation: metric.AggregationDrop{}},
		),
		metric.NewView(
			metric.Instrument{
				Name:  "otelcol_exporter_send_failed_metric_points",
				Scope: exporterHelperScope,
			},
			metric.Stream{Aggregation: metric.AggregationDrop{}},
		),
		metric.NewView(
			metric.Instrument{
				Name:  "otelcol_exporter_send_failed_spans",
				Scope: exporterHelperScope,
			},
			metric.Stream{Aggregation: metric.AggregationDrop{}},
		),
		metric.NewView(
			metric.Instrument{
				Name:  "otelcol_exporter_sent_log_records",
				Scope: exporterHelperScope,
			},
			metric.Stream{Aggregation: metric.AggregationDrop{}},
		),
		metric.NewView(
			metric.Instrument{
				Name:  "otelcol_exporter_sent_metric_points",
				Scope: exporterHelperScope,
			},
			metric.Stream{Aggregation: metric.AggregationDrop{}},
		),
		metric.NewView(
			metric.Instrument{
				Name:  "otelcol_exporter_sent_spans",
				Scope: exporterHelperScope,
			},
			metric.Stream{Aggregation: metric.AggregationDrop{}},
		),
		metric.NewView(
			metric.Instrument{
				Name:  "otelcol_exporter_queue_capacity",
				Scope: exporterHelperScope,
			},
			metric.Stream{Aggregation: metric.AggregationDrop{}},
		),
		metric.NewView(
			metric.Instrument{
				Name:  "otelcol_exporter_queue_size",
				Scope: exporterHelperScope,
			},
			metric.Stream{Aggregation: metric.AggregationDrop{}},
		),
	}

	// Create a new MeterProvider with the views applied
	meterProvider := metric.NewMeterProvider(metric.WithView(views...))

	// Return modified telemetry settings
	return component.TelemetrySettings{
		Logger:         original.Logger,
		TracerProvider: original.TracerProvider,
		MeterProvider:  meterProvider,
		Resource:       original.Resource,
	}
}

// newTracesProcessorWithExporterHelper creates a new traces processor using exporterhelper components.
func newTracesProcessorWithExporterHelper(set processor.Settings, nextConsumer consumer.Traces, cfg *Config) (processor.Traces, error) {
	set.Logger.Info("Creating traces processor with ExporterHelper")

	queueBatchConfig := translateToExporterHelperConfig(cfg)
	set.Logger.Info("Translated config to ExporterHelper config")

	// Use modified telemetry settings that rename exporter metrics to processor metrics
	modifiedTelemetrySettings := createProcessorTelemetrySettings(set.TelemetrySettings)

	exporterSet := exporter.Settings{
		ID:                set.ID,
		TelemetrySettings: modifiedTelemetrySettings,
		BuildInfo:         set.BuildInfo,
	}

	set.Logger.Info("About to call exporterhelper.NewTraces")
	result, err := exporterhelper.NewTraces(
		context.Background(),
		exporterSet,
		cfg,
		nextConsumer.ConsumeTraces,
		exporterhelper.WithQueue(queueBatchConfig),
	)
	if err != nil {
		set.Logger.Error("exporterhelper.NewTraces failed", zap.Error(err))
	} else {
		set.Logger.Info("exporterhelper.NewTraces completed successfully")
	}

	return result, err
}

// newMetricsProcessorWithExporterHelper creates a new metrics processor using exporterhelper components.
func newMetricsProcessorWithExporterHelper(set processor.Settings, nextConsumer consumer.Metrics, cfg *Config) (processor.Metrics, error) {
	queueBatchConfig := translateToExporterHelperConfig(cfg)

	// Use modified telemetry settings that rename exporter metrics to processor metrics
	modifiedTelemetrySettings := createProcessorTelemetrySettings(set.TelemetrySettings)

	exporterSet := exporter.Settings{
		ID:                set.ID,
		TelemetrySettings: modifiedTelemetrySettings,
		BuildInfo:         set.BuildInfo,
	}

	return exporterhelper.NewMetrics(
		context.Background(),
		exporterSet,
		cfg,
		nextConsumer.ConsumeMetrics,
		exporterhelper.WithQueue(queueBatchConfig),
	)
}

// newLogsProcessorWithExporterHelper creates a new logs processor using exporterhelper components.
func newLogsProcessorWithExporterHelper(set processor.Settings, nextConsumer consumer.Logs, cfg *Config) (processor.Logs, error) {
	queueBatchConfig := translateToExporterHelperConfig(cfg)

	// Use modified telemetry settings that rename exporter metrics to processor metrics
	modifiedTelemetrySettings := createProcessorTelemetrySettings(set.TelemetrySettings)

	exporterSet := exporter.Settings{
		ID:                set.ID,
		TelemetrySettings: modifiedTelemetrySettings,
		BuildInfo:         set.BuildInfo,
	}

	return exporterhelper.NewLogs(
		context.Background(),
		exporterSet,
		cfg,
		nextConsumer.ConsumeLogs,
		exporterhelper.WithQueue(queueBatchConfig),
	)
}
