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
	grpc_logsettable "github.com/grpc-ecosystem/go-grpc-middleware/logging/settable"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zapgrpc"

	"go.opentelemetry.io/collector/config"
)

var grpcSettableLogger grpc_logsettable.SettableLoggerV2

func init() {
	// Note: this needs to happen only once and that too before any gRPC calls.
	// The grpcSettableLogger.Set can be then be called to set any logger after in thread safe fashion.
	grpcSettableLogger = grpc_logsettable.ReplaceGrpcLoggerV2()
}

func NewLogger(cfg config.ServiceTelemetryLogs, options []zap.Option) (*zap.Logger, error) {
	// Copied from NewProductionConfig.
	zapCfg := &zap.Config{
		Level:       zap.NewAtomicLevelAt(cfg.Level),
		Development: cfg.Development,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         cfg.Encoding,
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
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

// SetGRPCLogger replaces grpc logger with logger cloned from baseLogger with exact configuration.
// The minimum level of gRPC logs is set to WARN should the loglevel of the collector is set to INFO to avoid
// copious logging from grpc framework.
func SetGRPCLogger(baseLogger *zap.Logger, loglevel zapcore.Level) {
	glogger := baseLogger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		var c zapcore.Core
		if loglevel == zap.InfoLevel {
			c, _ = zapcore.NewIncreaseLevelCore(core, zap.WarnLevel)
		} else {
			c = core
		}
		return c.With([]zapcore.Field{zap.Bool("grpc_log", true)})
	}))
	grpcSettableLogger.Set(zapgrpc.NewLogger(glogger))
}
