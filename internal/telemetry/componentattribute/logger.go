// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute // import "go.opentelemetry.io/collector/internal/telemetry/componentattribute"

import (
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ zapcore.Core = (*loggerCoreWithAttributes)(nil)

type loggerCoreWithAttributes struct {
	zapcore.Core
	from zapcore.Core
}

func LoggerWithAttributes(logger *zap.Logger, attrs attribute.Set) *zap.Logger {
	fields := []zap.Field{}
	for _, kv := range attrs.ToSlice() {
		fields = append(fields, zap.String(string(kv.Key), kv.Value.AsString()))
	}
	return logger.WithOptions(
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			if cwa, ok := core.(*loggerCoreWithAttributes); ok {
				core = cwa.from
			}
			return &loggerCoreWithAttributes{Core: core.With(fields), from: core}
		}),
	)
}
