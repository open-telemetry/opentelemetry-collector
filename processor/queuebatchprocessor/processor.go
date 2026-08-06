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

// newSignalProcessor builds the per-signal metrics and hands them to
// exporterhelper, which takes ownership of them only once it returns
// successfully. Shut them down when it does not.
func newSignalProcessor[P any](
	set processor.Settings,
	signal pipeline.Signal,
	create func(exporterhelper.ObsMetrics) (P, error),
) (P, error) {
	var zero P
	obsMetrics, err := newObsMetrics(set.TelemetrySettings, set.ID, signal)
	if err != nil {
		return zero, err
	}
	p, err := create(obsMetrics)
	if err != nil {
		obsMetrics.Shutdown()
		return zero, err
	}
	return p, nil
}

func newTracesProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Traces) (processor.Traces, error) {
	return newSignalProcessor(set, pipeline.SignalTraces, func(obsMetrics exporterhelper.ObsMetrics) (processor.Traces, error) {
		return exporterhelper.NewTraces(ctx, exporterSettings(set), cfg, next.ConsumeTraces, queueOptions(cfg, next.Capabilities(), obsMetrics)...)
	})
}

func newMetricsProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Metrics) (processor.Metrics, error) {
	return newSignalProcessor(set, pipeline.SignalMetrics, func(obsMetrics exporterhelper.ObsMetrics) (processor.Metrics, error) {
		return exporterhelper.NewMetrics(ctx, exporterSettings(set), cfg, next.ConsumeMetrics, queueOptions(cfg, next.Capabilities(), obsMetrics)...)
	})
}

func newLogsProcessor(ctx context.Context, set processor.Settings, cfg *Config, next consumer.Logs) (processor.Logs, error) {
	return newSignalProcessor(set, pipeline.SignalLogs, func(obsMetrics exporterhelper.ObsMetrics) (processor.Logs, error) {
		return exporterhelper.NewLogs(ctx, exporterSettings(set), cfg, next.ConsumeLogs, queueOptions(cfg, next.Capabilities(), obsMetrics)...)
	})
}

func newProfilesProcessor(ctx context.Context, set processor.Settings, cfg *Config, next xconsumer.Profiles) (xprocessor.Profiles, error) {
	return newSignalProcessor(set, xpipeline.SignalProfiles, func(obsMetrics exporterhelper.ObsMetrics) (xprocessor.Profiles, error) {
		return xexporterhelper.NewProfiles(ctx, exporterSettings(set), cfg, next.ConsumeProfiles, queueOptions(cfg, next.Capabilities(), obsMetrics)...)
	})
}
