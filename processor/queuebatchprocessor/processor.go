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

func queueOptions(cfg *Config) []exporterhelper.Option {
	return []exporterhelper.Option{
		exporterhelper.WithQueue(configoptional.Some(*cfg)),
		exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}),
	}
}

func newTracesProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Traces) (processor.Traces, error) {
	return exporterhelper.NewTraces(ctx, exporterSettings(set), cfg, next.ConsumeTraces, queueOptions(cfg)...)
}

func newMetricsProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Metrics) (processor.Metrics, error) {
	return exporterhelper.NewMetrics(ctx, exporterSettings(set), cfg, next.ConsumeMetrics, queueOptions(cfg)...)
}

func newLogsProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Logs) (processor.Logs, error) {
	return exporterhelper.NewLogs(ctx, exporterSettings(set), cfg, next.ConsumeLogs, queueOptions(cfg)...)
}

func newProfilesProcessor(ctx context.Context, set processor.Settings, cfg *Config, next xconsumer.Profiles) (xprocessor.Profiles, error) {
	return xexporterhelper.NewProfiles(ctx, exporterSettings(set), cfg, next.ConsumeProfiles, queueOptions(cfg)...)
}
