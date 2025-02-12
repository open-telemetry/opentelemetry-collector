// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute // import "go.opentelemetry.io/collector/internal/telemetry/componentattribute"

import (
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ zapcore.Core = (*coreWithout)(nil)

type coreWithout struct {
	zapcore.Core
	from   zapcore.Core
	fields []zap.Field
}

func NewLogger(logger *zap.Logger, attrs *attribute.Set) *zap.Logger {
	fields := []zap.Field{}
	for _, kv := range attrs.ToSlice() {
		fields = append(fields, zap.String(string(kv.Key), kv.Value.AsString()))
	}
	return logger.WithOptions(
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return &coreWithout{Core: core.With(fields), from: core, fields: fields}
		}),
	)
}

func (l *coreWithout) Without(keys ...string) zapcore.Core {
	excludeKeys := make(map[string]struct{})
	for _, key := range keys {
		excludeKeys[key] = struct{}{}
	}

	fieldsWithout := []zap.Field{}
	for _, field := range l.fields {
		if _, excluded := excludeKeys[field.Key]; !excluded {
			fieldsWithout = append(fieldsWithout, field)
		}
	}

	return &coreWithout{Core: l.from.With(fieldsWithout), from: l.from, fields: fieldsWithout}
}
