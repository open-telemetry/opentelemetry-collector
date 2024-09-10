// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package loggingexporter // import "go.opentelemetry.io/collector/exporter/loggingexporter"

import (
	"context"

	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/internal/common"
	"go.opentelemetry.io/collector/exporter/loggingexporter/internal/metadata"
)

// The value of "type" key in configuration.
var componentType = component.MustNewType("logging")

const (
	defaultSamplingInitial    = 2
	defaultSamplingThereafter = 500
)

// NewFactory creates a factory for Logging exporter
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		componentType,
		createDefaultConfig,
		exporter.WithTraces(createTracesExporter, metadata.TracesStability),
		exporter.WithMetrics(createMetricsExporter, metadata.MetricsStability),
		exporter.WithLogs(createLogsExporter, metadata.LogsStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		LogLevel:           zapcore.InfoLevel,
		Verbosity:          configtelemetry.LevelNormal,
		SamplingInitial:    defaultSamplingInitial,
		SamplingThereafter: defaultSamplingThereafter,
	}
}

func createTracesExporter(ctx context.Context, set exporter.Settings, config component.Config) (exporter.Traces, error) {
	set.TelemetrySettings.Logger.Warn("The logging exporter is DEPRECATED and will be REMOVED in v0.111.0. Use the debug exporter instead: https://github.com/open-telemetry/opentelemetry-collector/tree/main/exporter/debugexporter")
	cfg := config.(*Config)
	return common.CreateTracesExporter(ctx, set, config, &common.Common{
		Verbosity:          cfg.Verbosity,
		WarnLogLevel:       cfg.warnLogLevel,
		LogLevel:           cfg.LogLevel,
		SamplingInitial:    cfg.SamplingInitial,
		SamplingThereafter: cfg.SamplingThereafter,
	})
}

func createMetricsExporter(ctx context.Context, set exporter.Settings, config component.Config) (exporter.Metrics, error) {
	set.TelemetrySettings.Logger.Warn("The logging exporter is DEPRECATED and will be REMOVED in v0.111.0. Use the debug exporter instead: https://github.com/open-telemetry/opentelemetry-collector/tree/main/exporter/debugexporter")
	cfg := config.(*Config)
	return common.CreateMetricsExporter(ctx, set, config, &common.Common{
		Verbosity:          cfg.Verbosity,
		WarnLogLevel:       cfg.warnLogLevel,
		LogLevel:           cfg.LogLevel,
		SamplingInitial:    cfg.SamplingInitial,
		SamplingThereafter: cfg.SamplingThereafter,
	})
}

func createLogsExporter(ctx context.Context, set exporter.Settings, config component.Config) (exporter.Logs, error) {
	set.TelemetrySettings.Logger.Warn("The logging exporter is DEPRECATED and will be REMOVED in v0.111.0. Use the debug exporter instead: https://github.com/open-telemetry/opentelemetry-collector/tree/main/exporter/debugexporter")
	cfg := config.(*Config)
	return common.CreateLogsExporter(ctx, set, config, &common.Common{
		Verbosity:          cfg.Verbosity,
		WarnLogLevel:       cfg.warnLogLevel,
		LogLevel:           cfg.LogLevel,
		SamplingInitial:    cfg.SamplingInitial,
		SamplingThereafter: cfg.SamplingThereafter,
	})
}
