// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Telemetry struct {
	logger         *zap.Logger
	sampledLogger  *zap.Logger
	tracerProvider *sdktrace.TracerProvider
}

func (t *Telemetry) TracerProvider() trace.TracerProvider {
	return t.tracerProvider
}

func (t *Telemetry) Logger() *zap.Logger {
	return t.logger
}

func (t *Telemetry) SampledLogger() *zap.Logger {
	return t.sampledLogger
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	// TODO: Sync logger.
	return multierr.Combine(
		t.tracerProvider.Shutdown(ctx),
	)
}

// Settings holds configuration for building Telemetry.
type Settings struct {
	ZapOptions []zap.Option
}

// New creates a new Telemetry from Config.
func New(_ context.Context, set Settings, cfg Config) (*Telemetry, error) {
	logger, err := newLogger(cfg.Logs, set.ZapOptions)
	if err != nil {
		return nil, err
	}
	sampledLogger := newSampledLogger(cfg.Logs.Sampling, logger)

	tp := sdktrace.NewTracerProvider(
		// needed for supporting the zpages extension
		sdktrace.WithSampler(alwaysRecord()),
	)
	return &Telemetry{
		logger:         logger,
		sampledLogger:  sampledLogger,
		tracerProvider: tp,
	}, nil
}

func newLogger(cfg LogsConfig, options []zap.Option) (*zap.Logger, error) {
	// Copied from NewProductionConfig.
	zapCfg := &zap.Config{
		Level:             zap.NewAtomicLevelAt(cfg.Level),
		Development:       cfg.Development,
		Encoding:          cfg.Encoding,
		EncoderConfig:     zap.NewProductionEncoderConfig(),
		OutputPaths:       cfg.OutputPaths,
		ErrorOutputPaths:  cfg.ErrorOutputPaths,
		DisableCaller:     cfg.DisableCaller,
		DisableStacktrace: cfg.DisableStacktrace,
		InitialFields:     cfg.InitialFields,
	}

	if zapCfg.Encoding == "console" {
		// Human-readable timestamps for console format of logs.
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	logger, err := zapCfg.Build(options...)
	if err != nil {
		return nil, err
	}

	return logger, nil
}

func newSampledLogger(cfg *LogsSamplingConfig, logger *zap.Logger) *zap.Logger {
	if cfg == nil {
		cfg = newDefaultLogsSamplingConfig()
	}

	// Create a logger that samples all messages to "initial" per "tick" initially,
	// and cfg.Initial/cfg.Thereafter of messages after that.
	opts := zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewSamplerWithOptions(
			core,
			cfg.Tick,
			cfg.Initial,
			cfg.Thereafter,
		)
	})
	return logger.WithOptions(opts)
}

// newDefaultLogsSamplingConfig returns a default LogsSamplingConfig.
func newDefaultLogsSamplingConfig() *LogsSamplingConfig {
	return &LogsSamplingConfig{
		Initial:    1,
		Thereafter: 100,
		Tick:       10 * time.Second,
	}
}
