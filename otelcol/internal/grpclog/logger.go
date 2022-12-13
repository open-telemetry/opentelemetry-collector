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

package grpclog // import "go.opentelemetry.io/collector/otelcol/internal/grpclog"

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc/grpclog"
)

// SetLogger constructs a zapgrpc.Logger instance, and installs it as grpc logger, cloned from baseLogger with
// exact configuration. The minimum level of gRPC logs is set to WARN should the loglevel of the collector is set to
// INFO to avoid copious logging from grpc framework.
func SetLogger(baseLogger *zap.Logger, loglevel zapcore.Level) *zapgrpc.Logger {
	logger := zapgrpc.NewLogger(baseLogger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		var c zapcore.Core
		var err error
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
	})))

	grpclog.SetLoggerV2(logger)
	return logger
}
