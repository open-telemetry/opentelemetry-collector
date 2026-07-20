// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatchprocessor // import "go.opentelemetry.io/collector/processor/queuebatchprocessor"

import (
	"context"

	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/xprocessor"
)

func exporterSettings(set processor.Settings) exporter.Settings {
	return exporter.Settings{
		ID:                set.ID,
		TelemetrySettings: set.TelemetrySettings,
		BuildInfo:         set.BuildInfo,
	}
}

// queueOptions returns the exporterhelper options shared by every signal and
// derives the processor's MutatesData capability from the configuration and the
// next consumer. When batching is enabled the batcher merges and mutates
// requests, so the processor mutates data. When a persistent queue is configured
// the request is serialized on enqueue and the next consumer receives a freshly
// deserialized copy, so the caller's data is never mutated regardless of the
// next consumer. Otherwise the in-memory queue forwards the same pdata to the
// next consumer, so the processor mutates data exactly when the next consumer
// does.
func queueOptions(cfg *Config, next consumer.Capabilities) []exporterhelper.Option {
	var mutates bool
	if cfg.Batch.HasValue() {
		mutates = true
	} else if cfg.StorageID != nil {
		mutates = false
	} else {
		mutates = next.MutatesData
	}
	return []exporterhelper.Option{
		exporterhelper.WithQueue(configoptional.Some(*cfg)),
		exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}),
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: mutates}),
	}
}

func newTracesProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Traces) (processor.Traces, error) {
	return exporterhelper.NewTraces(ctx, exporterSettings(set), cfg, next.ConsumeTraces, queueOptions(cfg, next.Capabilities())...)
}

func newMetricsProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Metrics) (processor.Metrics, error) {
	return exporterhelper.NewMetrics(ctx, exporterSettings(set), cfg, next.ConsumeMetrics, queueOptions(cfg, next.Capabilities())...)
}

func newLogsProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Logs) (processor.Logs, error) {
	return exporterhelper.NewLogs(ctx, exporterSettings(set), cfg, next.ConsumeLogs, queueOptions(cfg, next.Capabilities())...)
}

func newProfilesProcessor(ctx context.Context, set processor.Settings, cfg *Config, next xconsumer.Profiles) (xprocessor.Profiles, error) {
	return xexporterhelper.NewProfiles(ctx, exporterSettings(set), cfg, next.ConsumeProfiles, queueOptions(cfg, next.Capabilities())...)
}
