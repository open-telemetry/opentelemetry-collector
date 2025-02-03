// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute // import "go.opentelemetry.io/collector/component/componentattribute"

import (
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ zapcore.Core = (*Core)(nil)

type Core struct {
	zapcore.Core
	from  *zap.Logger
	attrs attribute.Set
}

func NewLogger(from *zap.Logger, attrs *attribute.Set) *zap.Logger {
	withAttributes := from
	for _, kv := range attrs.ToSlice() {
		withAttributes = withAttributes.With(zap.String(string(kv.Key), kv.Value.AsString()))
	}
	return zap.New(&Core{
		Core:  withAttributes.Core(),
		from:  from,
		attrs: *attrs,
	})
}

func (l *Core) Without(keys ...string) *zap.Logger {
	excludeKeys := make(map[string]struct{})
	for _, key := range keys {
		excludeKeys[key] = struct{}{}
	}

	newAttrs := []attribute.KeyValue{}
	withAttributes := l.from
	for _, kv := range l.attrs.ToSlice() {
		if _, excluded := excludeKeys[string(kv.Key)]; !excluded {
			newAttrs = append(newAttrs, kv)
			withAttributes = withAttributes.With(zap.String(string(kv.Key), kv.Value.AsString()))
		}
	}

	return zap.New(&Core{
		Core:  withAttributes.Core(),
		from:  l.from,
		attrs: attribute.NewSet(newAttrs...),
	})
}
