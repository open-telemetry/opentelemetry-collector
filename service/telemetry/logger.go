// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

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

	switch cfg.TimeEncoding {
	case "Epoch":
		zapCfg.EncoderConfig.EncodeTime = zapcore.EpochTimeEncoder
	case "EpochMillis":
		zapCfg.EncoderConfig.EncodeTime = zapcore.EpochMillisTimeEncoder
	case "EpochNanos":
		zapCfg.EncoderConfig.EncodeTime = zapcore.EpochNanosTimeEncoder
	case "ISO8601":
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	case "RFC3339":
		zapCfg.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	case "RFC3339Nano":
		zapCfg.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	case "":
		// no-op
	}

	logger, err := zapCfg.Build(options...)
	if err != nil {
		return nil, err
	}
	if cfg.Sampling != nil && cfg.Sampling.Enabled {
		logger = newSampledLogger(logger, cfg.Sampling)
	}

	return logger, nil
}

func newSampledLogger(logger *zap.Logger, sc *LogsSamplingConfig) *zap.Logger {
	// Create a logger that samples every Nth message after the first M messages every S seconds
	// where N = sc.Thereafter, M = sc.Initial, S = sc.Tick.
	opts := zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewSamplerWithOptions(
			core,
			sc.Tick,
			sc.Initial,
			sc.Thereafter,
		)
	})
	return logger.WithOptions(opts)
}
