// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"go.opentelemetry.io/collector/internal/telemetry/componentattribute"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// newLogger creates a Logger and a LoggerProvider from Config.
func newLogger(set Settings, cfg Config) (*zap.Logger, log.LoggerProvider, error) {
	// Copied from NewProductionConfig.
	ec := zap.NewProductionEncoderConfig()
	ec.EncodeTime = zapcore.ISO8601TimeEncoder
	zapCfg := &zap.Config{
		Level:             zap.NewAtomicLevelAt(cfg.Logs.Level),
		Development:       cfg.Logs.Development,
		Encoding:          cfg.Logs.Encoding,
		EncoderConfig:     ec,
		OutputPaths:       cfg.Logs.OutputPaths,
		ErrorOutputPaths:  cfg.Logs.ErrorOutputPaths,
		DisableCaller:     cfg.Logs.DisableCaller,
		DisableStacktrace: cfg.Logs.DisableStacktrace,
		InitialFields:     cfg.Logs.InitialFields,
	}

	if zapCfg.Encoding == "console" {
		// Human-readable timestamps for console format of logs.
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	logger, err := zapCfg.Build(set.ZapOptions...)
	if err != nil {
		return nil, nil, err
	}

	var lp log.LoggerProvider

	if len(cfg.Logs.Processors) > 0 && set.SDK != nil {
		lp = set.SDK.LoggerProvider()

		logger = logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
			return componentattribute.NewServiceZapCore(lp, "go.opentelemetry.io/collector/service/telemetry", func(otelCore zapcore.Core, _ attribute.Set) zapcore.Core {
				// TODO: Add component attributes to the console output as well?
				core, err := zapcore.NewIncreaseLevelCore(zapcore.NewTee(
					c,
					otelCore,
				), zap.NewAtomicLevelAt(cfg.Logs.Level))
				if err != nil {
					panic(err)
				}
				if cfg.Logs.Sampling != nil && cfg.Logs.Sampling.Enabled {
					core = newSampledCore(core, cfg.Logs.Sampling)
				}
				return core
			}, attribute.NewSet())
		}))
	} else if cfg.Logs.Sampling != nil && cfg.Logs.Sampling.Enabled {
		logger = logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
			return newSampledCore(c, cfg.Logs.Sampling)
		}))
	}

	return logger, lp, nil
}

func newSampledCore(core zapcore.Core, sc *LogsSamplingConfig) zapcore.Core {
	// Create a logger that samples every Nth message after the first M messages every S seconds
	// where N = sc.Thereafter, M = sc.Initial, S = sc.Tick.
	return zapcore.NewSamplerWithOptions(
		core,
		sc.Tick,
		sc.Initial,
		sc.Thereafter,
	)
}
