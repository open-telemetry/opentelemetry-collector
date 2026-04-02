// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package grpclog // import "go.opentelemetry.io/collector/otelcol/internal/grpclog"

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc/grpclog"
)

// benignClientClosePatterns are gRPC transport-layer messages that occur during normal
// client disconnection and should not be surfaced as warnings to operators.
var benignClientClosePatterns = []string{
	"HandleStreams failed to read frame",
	"connection reset by peer",
}

func isBenignClientCloseMessage(msg string) bool {
	for _, pattern := range benignClientClosePatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}

// benignFilterCore wraps a zapcore.Core to suppress known benign gRPC client-disconnect
// warnings, reducing noise in production logs.
type benignFilterCore struct {
	zapcore.Core
}

func (c *benignFilterCore) With(fields []zapcore.Field) zapcore.Core {
	return &benignFilterCore{c.Core.With(fields)}
}

func (c *benignFilterCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if entry.Level == zapcore.WarnLevel && isBenignClientCloseMessage(entry.Message) {
		return checked
	}
	return c.Core.Check(entry, checked)
}

// SetLogger constructs a zapgrpc.Logger instance, and installs it as grpc logger, cloned from baseLogger with
// exact configuration. The minimum level of gRPC logs is set to WARN should the loglevel of the collector is set to
// INFO to avoid copious logging from grpc framework. Benign client-disconnect warnings are suppressed.
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
		return &benignFilterCore{c.With([]zapcore.Field{zap.Bool("grpc_log", true)})}
	}), zap.AddCallerSkip(5)))

	grpclog.SetLoggerV2(logger)
	return logger
}
