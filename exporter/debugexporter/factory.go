// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package debugexporter // import "go.opentelemetry.io/collector/exporter/debugexporter"

import (
	"context"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/debugexporter/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/internal/otlptext"
)

// The value of "type" key in configuration.
var componentType = component.MustNewType("debug")

const (
	defaultSamplingInitial    = 2
	defaultSamplingThereafter = 1
)

// NewFactory creates a factory for Debug exporter
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
		Verbosity:          configtelemetry.LevelBasic,
		SamplingInitial:    defaultSamplingInitial,
		SamplingThereafter: defaultSamplingThereafter,
		UseInternalLogger:  true,
	}
}

func createTracesExporter(ctx context.Context, set exporter.Settings, config component.Config) (exporter.Traces, error) {
	cfg := config.(*Config)
	exporterLogger := createLogger(cfg, set.TelemetrySettings.Logger)
	debugExporter := newDebugExporter(exporterLogger, cfg.Verbosity)
	return exporterhelper.NewTracesExporter(ctx, set, config,
		debugExporter.pushTraces,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithShutdown(otlptext.LoggerSync(exporterLogger)),
	)
}

func createMetricsExporter(ctx context.Context, set exporter.Settings, config component.Config) (exporter.Metrics, error) {
	cfg := config.(*Config)
	exporterLogger := createLogger(cfg, set.TelemetrySettings.Logger)
	debugExporter := newDebugExporter(exporterLogger, cfg.Verbosity)
	return exporterhelper.NewMetricsExporter(ctx, set, config,
		debugExporter.pushMetrics,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithShutdown(otlptext.LoggerSync(exporterLogger)),
	)
}

func createLogsExporter(ctx context.Context, set exporter.Settings, config component.Config) (exporter.Logs, error) {
	cfg := config.(*Config)
	exporterLogger := createLogger(cfg, set.TelemetrySettings.Logger)
	debugExporter := newDebugExporter(exporterLogger, cfg.Verbosity)
	return exporterhelper.NewLogsExporter(ctx, set, config,
		debugExporter.pushLogs,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithShutdown(otlptext.LoggerSync(exporterLogger)),
	)
}

func createLogger(cfg *Config, logger *zap.Logger) *zap.Logger {
	var exporterLogger *zap.Logger
	if cfg.UseInternalLogger {
		core := zapcore.NewSamplerWithOptions(
			logger.Core(),
			1*time.Second,
			cfg.SamplingInitial,
			cfg.SamplingThereafter,
		)
		exporterLogger = zap.New(core)
	} else {
		exporterLogger = createCustomLogger(cfg)
	}
	return exporterLogger
}

func createCustomLogger(exporterConfig *Config) *zap.Logger {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	// Do not prefix the output with log level (`info`)
	encoderConfig.LevelKey = ""
	// Do not prefix the output with current timestamp.
	encoderConfig.TimeKey = ""
	zapConfig := zap.Config{
		Level:         zap.NewAtomicLevelAt(zap.InfoLevel),
		DisableCaller: true,
		Sampling: &zap.SamplingConfig{
			Initial:    exporterConfig.SamplingInitial,
			Thereafter: exporterConfig.SamplingThereafter,
		},
		Encoding:      "console",
		EncoderConfig: encoderConfig,
		// Send exporter's output to stdout. This should be made configurable.
		OutputPaths: []string{"stdout"},
	}
	return zap.Must(zapConfig.Build())
}
