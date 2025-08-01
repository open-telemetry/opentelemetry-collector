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
		OutputPaths:        []string{"stdout"},
	}
}

func createTraces(ctx context.Context, set exporter.Settings, config component.Config) (exporter.Traces, error) {
	cfg := config.(*Config)
	exporterLogger, closer := createLogger(cfg, set.Logger)
	debug := newDebugExporter(exporterLogger, cfg.Verbosity)

	shutdownFunc := func(ctx context.Context) error {
		// First sync the logger
		if err := otlptext.LoggerSync(exporterLogger)(ctx); err != nil {
			return err
		}
		// Then close any opened files
		return closer()
	}

	return exporterhelper.NewTraces(ctx, set, config,
		debug.pushTraces,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}),
		exporterhelper.WithShutdown(shutdownFunc),
	)
}

func createMetrics(ctx context.Context, set exporter.Settings, config component.Config) (exporter.Metrics, error) {
	cfg := config.(*Config)
	exporterLogger, closer := createLogger(cfg, set.Logger)
	debug := newDebugExporter(exporterLogger, cfg.Verbosity)

	shutdownFunc := func(ctx context.Context) error {
		// First sync the logger
		if err := otlptext.LoggerSync(exporterLogger)(ctx); err != nil {
			return err
		}
		// Then close any opened files
		return closer()
	}

	return exporterhelper.NewMetrics(ctx, set, config,
		debug.pushMetrics,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}),
		exporterhelper.WithShutdown(shutdownFunc),
	)
}

func createLogs(ctx context.Context, set exporter.Settings, config component.Config) (exporter.Logs, error) {
	cfg := config.(*Config)
	exporterLogger, closer := createLogger(cfg, set.Logger)
	debug := newDebugExporter(exporterLogger, cfg.Verbosity)

	shutdownFunc := func(ctx context.Context) error {
		// First sync the logger
		if err := otlptext.LoggerSync(exporterLogger)(ctx); err != nil {
			return err
		}
		// Then close any opened files
		return closer()
	}

	return exporterhelper.NewLogs(ctx, set, config,
		debug.pushLogs,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}),
		exporterhelper.WithShutdown(shutdownFunc),
	)
}

func createProfiles(ctx context.Context, set exporter.Settings, config component.Config) (xexporter.Profiles, error) {
	cfg := config.(*Config)
	exporterLogger, closer := createLogger(cfg, set.Logger)
	debug := newDebugExporter(exporterLogger, cfg.Verbosity)

	shutdownFunc := func(ctx context.Context) error {
		// First sync the logger
		if err := otlptext.LoggerSync(exporterLogger)(ctx); err != nil {
			return err
		}
		// Then close any opened files
		return closer()
	}

	return xexporterhelper.NewProfiles(ctx, set, config,
		debug.pushProfiles,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}),
		exporterhelper.WithShutdown(shutdownFunc),
	)
}

// createLoggerFunc is a function variable that can be overridden in tests
var createLoggerFunc = createLoggerImpl

func createLogger(cfg *Config, logger *zap.Logger) (*zap.Logger, func() error) {
	return createLoggerFunc(cfg, logger)
}

func createLoggerImpl(cfg *Config, logger *zap.Logger) (*zap.Logger, func() error) {
	var exporterLogger *zap.Logger
	var closer func() error
	if cfg.UseInternalLogger {
		core := zapcore.NewSamplerWithOptions(
			logger.Core(),
			1*time.Second,
			cfg.SamplingInitial,
			cfg.SamplingThereafter,
		)
		exporterLogger = zap.New(core)
		closer = func() error { return nil } // No cleanup needed for internal logger
	} else {
		exporterLogger, closer = createCustomLogger(cfg)
	}
	return exporterLogger, closer
}

func createCustomLogger(exporterConfig *Config) (*zap.Logger, func() error) {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	// Do not prefix the output with log level (`info`)
	encoderConfig.LevelKey = ""
	// Do not prefix the output with current timestamp.
	encoderConfig.TimeKey = ""

	outputPaths := exporterConfig.OutputPaths

	// Open all output paths using zap.Open
	writers := make([]zapcore.WriteSyncer, 0, len(outputPaths))
	closers := make([]func(), 0, len(outputPaths))

	for _, path := range outputPaths {
		sink, closeFunc, err := zap.Open(path)
		if err != nil {
			// Close any previously opened sinks
			for _, closer := range closers {
				closer()
			}
			panic(err) // Maintain existing panic behavior similar to zap.Must
		}
		writers = append(writers, sink)
		if closeFunc != nil {
			closers = append(closers, closeFunc)
		}
	}

	// Create a multiwriter if there are multiple outputs
	var writeSyncer zapcore.WriteSyncer
	if len(writers) == 1 {
		writeSyncer = writers[0]
	} else {
		writeSyncer = zapcore.NewMultiWriteSyncer(writers...)
	}

	// Create the encoder
	encoder := zapcore.NewConsoleEncoder(encoderConfig)

	// Create the core with sampling
	core := zapcore.NewCore(encoder, writeSyncer, zap.InfoLevel)
	samplingCore := zapcore.NewSamplerWithOptions(
		core,
		1*time.Second,
		exporterConfig.SamplingInitial,
		exporterConfig.SamplingThereafter,
	)

	logger := zap.New(samplingCore)

	// Create a closer function that closes all opened files
	closer := func() error {
		for _, c := range closers {
			c() // Call the close function
		}
		return nil
	}

	return logger, closer
}
