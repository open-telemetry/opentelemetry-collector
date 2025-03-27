// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/internal/telemetry/componentattribute"
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

	if telemetry.NewPipelineTelemetryGate.IsEnabled() {
		logger = logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			core = componentattribute.NewConsoleCoreWithAttributes(core, attribute.NewSet())

			if len(cfg.Logs.Processors) > 0 && set.SDK != nil {
				lp = set.SDK.LoggerProvider()

				core = componentattribute.NewOTelTeeCoreWithAttributes(
					core,
					lp,
					"go.opentelemetry.io/collector/service/telemetry",
					cfg.Logs.Level,
					attribute.NewSet(),
				)
			}

			if cfg.Logs.Sampling != nil && cfg.Logs.Sampling.Enabled {
				core = componentattribute.NewWrapperCoreWithAttributes(core, func(c zapcore.Core) zapcore.Core {
					return newSampledCore(c, cfg.Logs.Sampling)
				})
			}

			return core
		}))
	} else {
		if len(cfg.Logs.Processors) > 0 && set.SDK != nil {
			lp = set.SDK.LoggerProvider()

			logger = logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
				core, err := zapcore.NewIncreaseLevelCore(zapcore.NewTee(
					c,
					otelzap.NewCore("go.opentelemetry.io/collector/service/telemetry",
						otelzap.WithLoggerProvider(lp),
					),
				), zap.NewAtomicLevelAt(cfg.Logs.Level))
				if err != nil {
					panic(err)
				}
				return core
			}))
		}

		if cfg.Logs.Sampling != nil && cfg.Logs.Sampling.Enabled {
			logger = logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
				return newSampledCore(c, cfg.Logs.Sampling)
			}))
		}
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
