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
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
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

// queueOptions returns the exporterhelper options shared by every signal.
func queueOptions(cfg *Config, next consumer.Capabilities, obsMetrics exporterhelper.ObsMetrics) []exporterhelper.Option {
	var mutates bool
	switch {
	case cfg.Batch.HasValue():
		mutates = true
	case cfg.StorageID != nil:
		mutates = false
	default:
		mutates = next.MutatesData
	}
	return []exporterhelper.Option{
		exporterhelper.WithObsMetrics(obsMetrics),
		exporterhelper.WithQueue(configoptional.Some(*cfg)),
		exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}),
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: mutates}),
	}
}

func newTracesProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Traces) (processor.Traces, error) {
	obsMetrics, err := newObsMetrics(set.TelemetrySettings, set.ID, pipeline.SignalTraces)
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewTraces(ctx, exporterSettings(set), cfg, next.ConsumeTraces, queueOptions(cfg, next.Capabilities(), obsMetrics)...)
}

func newMetricsProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Metrics) (processor.Metrics, error) {
	obsMetrics, err := newObsMetrics(set.TelemetrySettings, set.ID, pipeline.SignalMetrics)
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewMetrics(ctx, exporterSettings(set), cfg, next.ConsumeMetrics, queueOptions(cfg, next.Capabilities(), obsMetrics)...)
}

func newLogsProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Logs) (processor.Logs, error) {
	obsMetrics, err := newObsMetrics(set.TelemetrySettings, set.ID, pipeline.SignalLogs)
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewLogs(ctx, exporterSettings(set), cfg, next.ConsumeLogs, queueOptions(cfg, next.Capabilities(), obsMetrics)...)
}

func newProfilesProcessor(ctx context.Context, set processor.Settings, cfg *Config, next xconsumer.Profiles) (xprocessor.Profiles, error) {
	obsMetrics, err := newObsMetrics(set.TelemetrySettings, set.ID, xpipeline.SignalProfiles)
	if err != nil {
		return nil, err
	}
	return xexporterhelper.NewProfiles(ctx, exporterSettings(set), cfg, next.ConsumeProfiles, queueOptions(cfg, next.Capabilities(), obsMetrics)...)
}
