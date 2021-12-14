// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package telemetrylogs // import "go.opentelemetry.io/collector/service/internal/telemetrylogs"

import (
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc/grpclog"

	"go.opentelemetry.io/collector/config"
)

func NewLogger(cfg config.ServiceTelemetryLogs, options []zap.Option) (*zap.Logger, error) {
	// Copied from NewProductionConfig.
	zapCfg := &zap.Config{
		Level:       zap.NewAtomicLevelAt(cfg.Level),
		Development: cfg.Development,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:          cfg.Encoding,
		EncoderConfig:     zap.NewProductionEncoderConfig(),
		OutputPaths:       cfg.OutputPaths,
		ErrorOutputPaths:  cfg.ErrorOutputPaths,
		DisableCaller:     cfg.DisableCaller,
		DisableStacktrace: cfg.DisableStacktrace,
		InitialFields:     cfg.InitialFields,
	}

	if zapCfg.Encoding == "console" {
		// Human-readable timestamps for console format of logs.
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	logger, err := zapCfg.Build(options...)
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// SettableGRPCLoggerV2 sets grpc framework's logger with internal logger.
type SettableGRPCLoggerV2 interface {
	SetGRPCLogger()
}

type colGRPCLogger struct {
	setOnce  sync.Once
	loggerV2 grpclog.LoggerV2
}

// NewColGRPCLogger constructs a grpclog.LoggerV2 instance cloned from baseLogger with exact configuration.
// The minimum level of gRPC logs is set to WARN should the loglevel of the collector is set to INFO to avoid
// copious logging from grpc framework.
func NewColGRPCLogger(baseLogger *zap.Logger, loglevel zapcore.Level) SettableGRPCLoggerV2 {
	logger := baseLogger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		var c zapcore.Core
		if loglevel == zap.InfoLevel {
			// NewIncreaseLevelCore errors only if the new log level is less than the initial core level.
			// In this case it never happens as WARN is always greater than INFO, therefore ignoring it.
			c, _ = zapcore.NewIncreaseLevelCore(core, zap.WarnLevel)
		} else {
			c = core
		}
		return c.With([]zapcore.Field{zap.Bool("grpc_log", true)})
	}))
	return &colGRPCLogger{
		loggerV2: zapgrpc.NewLogger(logger),
	}
}

// SetGRPCLogger needs to be run before any grpc calls and this implementation requires it to be run
// only once.
func (gl *colGRPCLogger) SetGRPCLogger() {
	gl.setOnce.Do(func() {
		grpclog.SetLoggerV2(gl.loggerV2)
	})
}
