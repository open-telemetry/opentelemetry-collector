// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"
	"sort"

	otelconf "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/telemetry"
)

// createLogger creates a Logger and a LoggerProvider from Config.
func (f *otelconfFactory) createLogger(
	ctx context.Context,
	set telemetry.LoggerSettings,
	componentConfig component.Config,
) (*zap.Logger, component.ShutdownFunc, error) {
	cfg := componentConfig.(*Config)

	resourceConfig, err := createResourceConfig(ctx, set.BuildInfo, &cfg.Resource, set.Resource)
	if err != nil {
		return nil, nil, err
	}
	res := set.Resource

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

	// The attributes in res.Attributes(), which are generated in resource.go,
	// are added to logs exported through the LoggerProvider instantiated below.
	// To make sure they are also exposed in logs written to stdout, we add
	// them as fields to the Zap core created above using WrapCore. We do NOT
	// add them to the logger using With, because that would apply to all logs,
	// even ones exported through the core that wraps the LoggerProvider,
	// meaning that the attributes would be exported twice.
	if !cfg.Logs.DisableZapResource && res != nil && res.Attributes().Len() > 0 {
		logger = logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
			var fields []zap.Field
			for key, val := range res.Attributes().All() {
				fields = append(fields, zap.String(key, val.AsString()))
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

	sdk, err := newSDK(ctx, resourceConfig, otelconf.OpenTelemetryConfiguration{
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
	warnLegacyResourceAttributes(logger, cfg)

	return logger, sdk.Shutdown, nil
}

func warnLegacyResourceAttributes(logger *zap.Logger, cfg *Config) {
	if len(cfg.Resource.LegacyAttributes) == 0 {
		return
	}
	legacyKeys := make([]string, 0, len(cfg.Resource.LegacyAttributes))
	for key := range cfg.Resource.LegacyAttributes {
		legacyKeys = append(legacyKeys, key)
	}
	sort.Strings(legacyKeys)
	logger.Warn(
		"Using legacy service.telemetry.resource inline map format; prefer service.telemetry.resource.attributes",
		zap.Strings("legacy_resource_attributes", legacyKeys),
	)
}
