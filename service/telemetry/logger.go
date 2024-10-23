// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/contrib/config"
	"go.opentelemetry.io/otel/log"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newLogger(ctx context.Context, set Settings, cfg Config) (*zap.Logger, log.LoggerProvider, error) {
	// Copied from NewProductionConfig.
	zapCfg := &zap.Config{
		Level:             zap.NewAtomicLevelAt(cfg.Logs.Level),
		Development:       cfg.Logs.Development,
		Encoding:          cfg.Logs.Encoding,
		EncoderConfig:     zap.NewProductionEncoderConfig(),
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

	if len(cfg.Logs.Processors) > 0 {
		sch := semconv.SchemaURL
		res := config.Resource{
			SchemaUrl:  &sch,
			Attributes: attributes(set, cfg),
		}
		sdk, err := config.NewSDK(
			config.WithContext(ctx),
			config.WithOpenTelemetryConfiguration(
				config.OpenTelemetryConfiguration{
					LoggerProvider: &config.LoggerProvider{
						Processors: cfg.Logs.Processors,
					},
					Resource: &res,
				},
			),
		)

		if err != nil {
			return nil, nil, err
		}

		lp = sdk.LoggerProvider()

		logger = logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
			return zapcore.NewTee(
				c,
				otelzap.NewCore("go.opentelemetry.io/collector/service/telemetry",
					otelzap.WithLoggerProvider(lp),
				),
			)
		}))

	}

	if cfg.Logs.Sampling != nil && cfg.Logs.Sampling.Enabled {
		logger = newSampledLogger(logger, cfg.Logs.Sampling)
	}

	return logger, lp, nil
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
