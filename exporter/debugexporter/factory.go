// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package debugexporter // import "go.opentelemetry.io/collector/exporter/debugexporter"

import (
	"context"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/debugexporter/internal/metadata"
	"go.opentelemetry.io/collector/exporter/debugexporter/internal/otlptext"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper"
	"go.opentelemetry.io/collector/exporter/xexporter"
)

// The value of "type" key in configuration.
var componentType = component.MustNewType("debug")

const (
	defaultSamplingInitial    = 2
	defaultSamplingThereafter = 1
)

// NewFactory creates and returns a new factory for the Debug exporter.
func NewFactory() exporter.Factory {
	return xexporter.NewFactory(
		componentType,
		createDefaultConfig,
		xexporter.WithTraces(createTraces, metadata.TracesStability),
		xexporter.WithMetrics(createMetrics, metadata.MetricsStability),
		xexporter.WithLogs(createLogs, metadata.LogsStability),
		xexporter.WithProfiles(createProfiles, metadata.ProfilesStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		Verbosity:          configtelemetry.LevelBasic,
		SamplingInitial:    defaultSamplingInitial,
		SamplingThereafter: defaultSamplingThereafter,
		UseInternalLogger:  true,
		QueueConfig:        configoptional.Default(exporterhelper.NewDefaultQueueConfig()),
	}
}

func createTraces(ctx context.Context, set exporter.Settings, config component.Config) (exporter.Traces, error) {
	cfg := config.(*Config)
	exporterLogger := createLogger(cfg, set.Logger)
	debug := newDebugExporter(exporterLogger, cfg.Verbosity)
	return exporterhelper.NewTraces(ctx, set, config,
		debug.pushTraces,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithQueue(cfg.QueueConfig),
		exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}),
		exporterhelper.WithShutdown(otlptext.LoggerSync(exporterLogger)),
	)
}

func createMetrics(ctx context.Context, set exporter.Settings, config component.Config) (exporter.Metrics, error) {
	cfg := config.(*Config)
	exporterLogger := createLogger(cfg, set.Logger)
	debug := newDebugExporter(exporterLogger, cfg.Verbosity)
	return exporterhelper.NewMetrics(ctx, set, config,
		debug.pushMetrics,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithQueue(cfg.QueueConfig),
		exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}),
		exporterhelper.WithShutdown(otlptext.LoggerSync(exporterLogger)),
	)
}

func createLogs(ctx context.Context, set exporter.Settings, config component.Config) (exporter.Logs, error) {
	cfg := config.(*Config)
	exporterLogger := createLogger(cfg, set.Logger)
	debug := newDebugExporter(exporterLogger, cfg.Verbosity)
	return exporterhelper.NewLogs(ctx, set, config,
		debug.pushLogs,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithQueue(cfg.QueueConfig),
		exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}),
		exporterhelper.WithShutdown(otlptext.LoggerSync(exporterLogger)),
	)
}

func createProfiles(ctx context.Context, set exporter.Settings, config component.Config) (xexporter.Profiles, error) {
	cfg := config.(*Config)
	exporterLogger := createLogger(cfg, set.Logger)
	debug := newDebugExporter(exporterLogger, cfg.Verbosity)
	return xexporterhelper.NewProfiles(ctx, set, config,
		debug.pushProfiles,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithQueue(cfg.QueueConfig),
		exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}),
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
