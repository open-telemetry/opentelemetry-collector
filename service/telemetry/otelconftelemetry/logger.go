// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"

	otelconf "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/telemetry"
)

// createLogger creates a Logger and a LoggerProvider from Config.
func createLogger(
	ctx context.Context,
	set telemetry.LoggerSettings,
	componentConfig component.Config,
) (*zap.Logger, component.ShutdownFunc, error) {
	cfg := componentConfig.(*Config)
	attrs := pcommonAttrsToOTelAttrs(set.Resource)
	res := sdkresource.NewWithAttributes("", attrs...)

	// Copied from NewProductionConfig.
	ec := zap.NewProductionEncoderConfig()
	ec.EncodeTime = zapcore.ISO8601TimeEncoder
	zapCfg := zap.Config{
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

	buildZapLogger := set.BuildZapLogger
	if buildZapLogger == nil {
		// For backwards compatibility, use zap.Config.Build
		// if set.BuildZapLogger is not provided.
		buildZapLogger = zap.Config.Build
	}
	logger, err := buildZapLogger(zapCfg, set.ZapOptions...)
	if err != nil {
		return nil, nil, err
	}

	// The attributes in res.Attributes(), which are generated in telemetry.go,
	// are added to logs exported through the LoggerProvider instantiated below.
	// To make sure they are also exposed in logs written to stdout, we add
	// them as fields to the Zap core created above using WrapCore. We do NOT
	// add them to the logger using With, because that would apply to all logs,
	// even ones exported through the core that wraps the LoggerProvider,
	// meaning that the attributes would be exported twice.
	if !cfg.Logs.DisableZapResource && res != nil && len(res.Attributes()) > 0 {
		logger = logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
			var fields []zap.Field
			for _, attr := range res.Attributes() {
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

	sdk, err := newSDK(ctx, res, otelconf.OpenTelemetryConfiguration{
		LoggerProvider: &otelconf.LoggerProvider{
			Processors: cfg.Logs.Processors,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	// Wrap the zap.Logger with a special zap.Core so scope attributes
	// can be added and removed dynamically, and tee logs to the
	// LoggerProvider.
	loggerProvider := sdk.LoggerProvider()
	logger = logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		provider := zapCoreProvider{
			sourceCore: core,
			lp:         loggerProvider,
			scopeName:  "go.opentelemetry.io/collector/service",
		}
		return provider.newCore()
	}))

	return logger, sdk.Shutdown, nil
}
