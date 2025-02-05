// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute // import "go.opentelemetry.io/collector/component/componentattribute"

import (
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ zapcore.Core = (*coreWithout)(nil)

type coreWithout struct {
	zapcore.Core
	from  *zap.Logger
	attrs attribute.Set
}

func NewLogger(from *zap.Logger, attrs *attribute.Set) *zap.Logger {
	fields := []zap.Field{}
	for _, kv := range attrs.ToSlice() {
		fields = append(fields, zap.String(string(kv.Key), kv.Value.AsString()))
	}
	return from.WithOptions(
		zap.Fields(fields...),
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return &coreWithout{Core: core, from: from, attrs: *attrs}
		}),
	)
}

func (l *coreWithout) Without(keys ...string) *zap.Logger {
	excludeKeys := make(map[string]struct{})
	for _, key := range keys {
		excludeKeys[key] = struct{}{}
	}

	newAttrs := []attribute.KeyValue{}
	fields := []zap.Field{}
	for _, kv := range l.attrs.ToSlice() {
		if _, excluded := excludeKeys[string(kv.Key)]; !excluded {
			newAttrs = append(newAttrs, kv)
			fields = append(fields, zap.String(string(kv.Key), kv.Value.AsString()))
		}
	}

	return l.from.WithOptions(
		zap.Fields(fields...),
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return &coreWithout{Core: core, from: l.from, attrs: attribute.NewSet(newAttrs...)}
		}),
	)
}
