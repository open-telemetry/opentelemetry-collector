// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"net/url"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"

	"go.opentelemetry.io/collector/internal/telemetry/componentattribute"
)

var (
	rotationSchema string
	ljLogger       *lumberjack.Logger
)

type logRotateSink struct {
	*lumberjack.Logger
}

// Sync is a no-op method to satisfy the zap.Sink interface, ensuring compatibility with zap's logging framework.
func (lr logRotateSink) Sync() error {
	// no-op
	return nil
}

// GetRotatedLogger returns the global lumberjack logger instance.
func GetRotatedLogger() *lumberjack.Logger {
	return ljLogger
}

func createLumberjackLogger(logsCfg LogsConfig, logFileName string) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   logFileName,
		MaxSize:    logsCfg.Rotation.MaxMegabytes,
		MaxAge:     logsCfg.Rotation.MaxAge,
		MaxBackups: logsCfg.Rotation.MaxBackups,
		Compress:   logsCfg.Rotation.Compress,
	}
}

// getFirstFileOutputPath iterates through the output paths and returns the first
// path that is not a reserved keyword (console, stdout, stderr).
// It returns an empty string if no suitable file path is found.
func getFirstFileOutputPath(logsCfg LogsConfig) (string, int) {
	if len(logsCfg.OutputPaths) == 0 {
		return "", -1
	}

	for fIdx, path := range logsCfg.OutputPaths {
		switch path {
		case "console", "stdout", "stderr":
			// Ignore these keywords
			continue
		default:
			// This is considered a file path
			return path, fIdx
		}
	}
	// No file path found
	return "", -1
}

func registerLumberjackSink(logger *lumberjack.Logger, rotationSchemaLocal string) error {
	err := zap.RegisterSink(rotationSchemaLocal, func(*url.URL) (zap.Sink, error) {
		return logRotateSink{Logger: logger}, nil
	})
	return err
}

// newLogger creates a Logger and a LoggerProvider from Config.
// It generates a unique rotation schema if log rotation is enabled, then calls makeLogger.
// This separation facilitates testing makeLogger's lumberjack sink
// registration error path by allowing a predictable schema in tests.
func newLogger(set Settings, cfg Config) (*zap.Logger, log.LoggerProvider, error) {
	rotationSchema = ""
	if cfg.Logs.Rotation != nil && cfg.Logs.Rotation.Enabled {
		rotationSchema = "lumberjack-" + uuid.NewString()
	}

	return makeLogger(set, cfg, rotationSchema)
}

// makeLogger creates a Logger and a LoggerProvider from Config and a custom rotation schema.
func makeLogger(set Settings, cfg Config, rotationSchema string) (*zap.Logger, log.LoggerProvider, error) {
	logFileName, logFileIndex := getFirstFileOutputPath(cfg.Logs)

	if cfg.Logs.Rotation != nil && cfg.Logs.Rotation.Enabled && len(logFileName) > 0 {
		ljLogger = createLumberjackLogger(cfg.Logs, logFileName)

		err := registerLumberjackSink(ljLogger, rotationSchema)
		if err != nil {
			return nil, nil, err
		}
		cfg.Logs.OutputPaths[logFileIndex] = rotationSchema + ":" + logFileName
	}

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

	logger, lp := configureLogger(logger, cfg, set)

	return logger, lp, nil
}

// configureLogger applies common configuration to the logger
func configureLogger(logger *zap.Logger, cfg Config, set Settings) (*zap.Logger, log.LoggerProvider) {
	// The attributes in cfg.Resource are added as resource attributes for logs exported through the LoggerProvider instantiated below.
	// To make sure they are also exposed in logs written to stdout, we add them as fields to the Zap core created above using WrapCore.
	// We do NOT add them to the logger using With, because that would apply to all logs, even ones exported through the core that wraps
	// the LoggerProvider, meaning that the attributes would be exported twice.
	logger = logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		var fields []zap.Field
		for k, v := range cfg.Resource {
			if v != nil {
				f := zap.Stringp(k, v)
				fields = append(fields, f)
			}
		}
		r := zap.Dict("resource", fields...)
		return c.With([]zapcore.Field{r})
	}))

	var lp log.LoggerProvider
	logger = logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		if cfg.Logs.Sampling != nil && cfg.Logs.Sampling.Enabled {
			core = newSampledCore(core, cfg.Logs.Sampling)
		}

		core = componentattribute.NewConsoleCoreWithAttributes(core, attribute.NewSet())

		if len(cfg.Logs.Processors) > 0 && set.SDK != nil {
			lp = set.SDK.LoggerProvider()
			wrapper := func(c zapcore.Core) zapcore.Core {
				return c
			}
			if cfg.Logs.Sampling != nil && cfg.Logs.Sampling.Enabled {
				wrapper = func(c zapcore.Core) zapcore.Core {
					return newSampledCore(c, cfg.Logs.Sampling)
				}
			}

			core = componentattribute.NewOTelTeeCoreWithAttributes(
				core,
				lp,
				"go.opentelemetry.io/collector/service/telemetry",
				cfg.Logs.Level,
				attribute.NewSet(),
				wrapper,
			)
		}

		return core
	}))

	return logger, lp
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
