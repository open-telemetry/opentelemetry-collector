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
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

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

	grpc_zap.ReplaceGrpcLoggerV2(newGRPCLogger(logger, cfg.Level))
	return logger, nil
}

func newGRPCLogger(logger *zap.Logger, loglevel zapcore.Level) *zap.Logger {
	glogger := logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		if loglevel == zap.InfoLevel {
			c, _ := zapcore.NewIncreaseLevelCore(core, loglevel+1)
			return c
		}
		return core
	}))
	return glogger
}
