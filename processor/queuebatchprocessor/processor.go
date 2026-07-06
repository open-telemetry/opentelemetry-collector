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

// exporterSettings derives exporter.Settings from processor.Settings so the
// processor can build on the exporterhelper queue/batch implementation. The
// telemetry emitted by the exporterhelper is currently reported under its own
// exporter-namespaced names and scope; reframing it as processor telemetry is
// deferred to follow-up work.
func exporterSettings(set processor.Settings) exporter.Settings {
	return exporter.Settings{
		ID:                set.ID,
		TelemetrySettings: set.TelemetrySettings,
		BuildInfo:         set.BuildInfo,
	}
}

// queueOptions are the exporterhelper options shared by all signals: the
// queue/batch configuration and a disabled per-export timeout. The batch
// processor never imposed an export timeout, so neither does this processor;
// callers that want one can wrap the next consumer.
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
