// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

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
			return newSampledCore(core, cfg.Logs.Sampling)
		}))
	}

	var lp log.LoggerProvider
	if set.SDK != nil {
		// Make sure the returned LoggerProvider filters logs based on
		// the configured zap logger level. Otherwise the tee core that
		// the service package creates will increase the level.
		lp = &minsevLoggerProvider{
			LoggerProvider: set.SDK.LoggerProvider(),
			severity:       convertLevel(logger.Level()),
		}
	} else {
		lp = noop.NewLoggerProvider()
	}
	return logger, lp, nil
}

func newSampledCore(core zapcore.Core, sc *LogsSamplingConfig) zapcore.Core {
	// Create a logger that samples every Nth message after the first M messages every S seconds
	// where N = sc.Thereafter, M = sc.Initial, S = sc.Tick.
	return componentattribute.NewSamplerCoreWithAttributes(
		core,
		sc.Tick,
		sc.Initial,
		sc.Thereafter,
	)
}

// minsevLoggerProvider wraps a LoggerProvider, and filters logs based on the
// configured zap logger level. Ideally we would use the opentelemetry-go-contrib
// minsev processor, but it is not currently possible to pass in processors when
// constructing a LoggerProvider via otelconf.
type minsevLoggerProvider struct {
	log.LoggerProvider
	severity log.Severity
}

func (p *minsevLoggerProvider) Logger(name string, options ...log.LoggerOption) log.Logger {
	logger := p.LoggerProvider.Logger(name, options...)
	return &minsevLogger{Logger: logger, severity: p.severity}
}

type minsevLogger struct {
	log.Logger
	severity log.Severity
}

func (l *minsevLogger) Enabled(ctx context.Context, param log.EnabledParameters) bool {
	if param.Severity < l.severity {
		return false
	}
	return l.Logger.Enabled(ctx, param)
}

// Copied from otelzap
func convertLevel(level zapcore.Level) log.Severity {
	switch level {
	case zapcore.DebugLevel:
		return log.SeverityDebug
	case zapcore.InfoLevel:
		return log.SeverityInfo
	case zapcore.WarnLevel:
		return log.SeverityWarn
	case zapcore.ErrorLevel:
		return log.SeverityError
	case zapcore.DPanicLevel:
		return log.SeverityFatal1
	case zapcore.PanicLevel:
		return log.SeverityFatal2
	case zapcore.FatalLevel:
		return log.SeverityFatal3
	default:
		return log.SeverityUndefined
	}
}
