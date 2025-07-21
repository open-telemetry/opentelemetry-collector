// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
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

	// The attributes in set.Resource.Attributes(), which are generated in service.go, are added
	// as resource attributes for logs exported through the LoggerProvider instantiated below.
	// To make sure they are also exposed in logs written to stdout, we add them as fields to the
	// Zap core created above using WrapCore.
	// We do NOT add them to the logger using With, because that would apply to all logs, even ones
	// exported through the core that wraps the LoggerProvider, meaning that the attributes would
	// be exported twice.
	if set.Resource != nil && len(set.Resource.Attributes()) > 0 {
		logger = logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
			var fields []zap.Field
			for _, attr := range set.Resource.Attributes() {
				fields = append(fields, zap.String(string(attr.Key), attr.Value.Emit()))
			}

			r := zap.Dict("resource", fields...)
			return c.With([]zapcore.Field{r})
		}))
	}

	if cfg.Logs.Sampling != nil && cfg.Logs.Sampling.Enabled {
		logger = logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewSamplerWithOptions(
				core,
				cfg.Logs.Sampling.Tick,
				cfg.Logs.Sampling.Initial,
				cfg.Logs.Sampling.Thereafter,
			)
		}))
	}

	var lp log.LoggerProvider
	if set.SDK != nil {
		lp = set.SDK.LoggerProvider()
	} else {
		lp = noop.NewLoggerProvider()
	}
	return logger, lp, nil
}
