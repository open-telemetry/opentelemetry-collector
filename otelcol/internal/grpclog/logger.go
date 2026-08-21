// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package grpclog // import "go.opentelemetry.io/collector/otelcol/internal/grpclog"

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc/grpclog"
)

// fixedVerbosityLogger wraps a grpclog.LoggerV2 and reports verbosity against a
// fixed threshold instead of zap's severity enabler.
//
// zapgrpc.Logger.V(level) is implemented as levelEnabler.Enabled(zapcore.Level(level-1)),
// which conflates grpclog's supplemental verbosity with zap's severity. With WARN
// enabled, V(2) returns true, so grpc-go emits chatty per-RPC messages — including
// transport-layer notices logged on normal client disconnect — at WARN. See
// uber-go/zap#1544.
//
// Comparing against a fixed threshold restores grpclog's intended semantics.
// The default of 0 matches grpclog when GRPC_GO_LOG_VERBOSITY_LEVEL is unset.
type fixedVerbosityLogger struct {
	grpclog.LoggerV2
	verbosity int
}

func (l *fixedVerbosityLogger) V(level int) bool {
	return level <= l.verbosity
}

// SetLogger constructs a zapgrpc.Logger instance, and installs it as grpc logger, cloned from baseLogger with
// exact configuration. The minimum level of gRPC logs is set to WARN should the loglevel of the collector is set to
// INFO to avoid copious logging from grpc framework.
func SetLogger(baseLogger *zap.Logger) *zapgrpc.Logger {
	logger := zapgrpc.NewLogger(baseLogger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		var c zapcore.Core
		var err error
		loglevel := baseLogger.Level()
		if loglevel == zapcore.InfoLevel {
			loglevel = zapcore.WarnLevel
		}
		// NewIncreaseLevelCore errors only if the new log level is less than the initial core level.
		c, err = zapcore.NewIncreaseLevelCore(core, loglevel)
		// In case of an error changing the level, move on, this happens when using the NopCore
		if err != nil {
			c = core
		}
		return c.With([]zapcore.Field{zap.Bool("grpc_log", true)})
	}), zap.AddCallerSkip(5)))

	grpclog.SetLoggerV2(&fixedVerbosityLogger{LoggerV2: logger})
	return logger
}
