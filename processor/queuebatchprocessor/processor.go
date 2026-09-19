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
	queuebatchtelemetry "go.opentelemetry.io/collector/internal/telemetry/queuebatch"
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
func queueOptions(cfg *Config, next consumer.Capabilities) []exporterhelper.Option {
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
		exporterhelper.WithQueue(configoptional.Some(*cfg)),
		exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}),
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: mutates}),
	}
}

func newProcessor[P any](
	set processor.Settings,
	signal pipeline.Signal,
	sizer exporterhelper.RequestSizerType,
	create func(queuebatchtelemetry.ObsMetrics) (P, error),
) (P, error) {
	obsMetrics, err := newObsMetrics(set.TelemetrySettings, set.ID, signal, sizer)
	if err != nil {
		var zero P
		return zero, err
	}
	p, err := create(obsMetrics)
	if err != nil {
		obsMetrics.Shutdown()
	}
	return p, err
}

func newTracesProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Traces) (processor.Traces, error) {
	return newProcessor(set, pipeline.SignalTraces, cfg.Sizer, func(obsMetrics queuebatchtelemetry.ObsMetrics) (processor.Traces, error) {
		wrappedCfg := queuebatchtelemetry.ConfigWithObsMetrics(cfg, obsMetrics)
		return exporterhelper.NewTraces(ctx, exporterSettings(set), wrappedCfg, next.ConsumeTraces, queueOptions(cfg, next.Capabilities())...)
	})
}

func newMetricsProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Metrics) (processor.Metrics, error) {
	return newProcessor(set, pipeline.SignalMetrics, cfg.Sizer, func(obsMetrics queuebatchtelemetry.ObsMetrics) (processor.Metrics, error) {
		wrappedCfg := queuebatchtelemetry.ConfigWithObsMetrics(cfg, obsMetrics)
		return exporterhelper.NewMetrics(ctx, exporterSettings(set), wrappedCfg, next.ConsumeMetrics, queueOptions(cfg, next.Capabilities())...)
	})
}

func newLogsProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Logs) (processor.Logs, error) {
	return newProcessor(set, pipeline.SignalLogs, cfg.Sizer, func(obsMetrics queuebatchtelemetry.ObsMetrics) (processor.Logs, error) {
		wrappedCfg := queuebatchtelemetry.ConfigWithObsMetrics(cfg, obsMetrics)
		return exporterhelper.NewLogs(ctx, exporterSettings(set), wrappedCfg, next.ConsumeLogs, queueOptions(cfg, next.Capabilities())...)
	})
}

func newProfilesProcessor(ctx context.Context, set processor.Settings, cfg *Config, next xconsumer.Profiles) (xprocessor.Profiles, error) {
	return newProcessor(set, xpipeline.SignalProfiles, cfg.Sizer, func(obsMetrics queuebatchtelemetry.ObsMetrics) (xprocessor.Profiles, error) {
		wrappedCfg := queuebatchtelemetry.ConfigWithObsMetrics(cfg, obsMetrics)
		return xexporterhelper.NewProfiles(ctx, exporterSettings(set), wrappedCfg, next.ConsumeProfiles, queueOptions(cfg, next.Capabilities())...)
	})
}
