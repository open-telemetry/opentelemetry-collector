// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute // import "go.opentelemetry.io/collector/service/internal/componentattribute"

import (
	"slices"

	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/internal/telemetry"
)

// This wrapper around zapcore.Field tells the Zap -> OTel bridge that the field
// should be turned into an instrumentation scope instead of a set of log record attributes.
type ScopeAttributesField struct {
	Fields []zapcore.Field
	Attrs  []attribute.KeyValue
}

var _ zapcore.ObjectMarshaler = ScopeAttributesField{}

func (saf ScopeAttributesField) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for _, field := range saf.Fields {
		field.AddTo(enc)
	}
	return nil
}

func makeScopeField(attrs []attribute.KeyValue) zap.Field {
	return zap.Inline(ScopeAttributesField{
		Fields: telemetry.ToZapFields(attrs),
		Attrs:  attrs,
	})
}

type coreWithAttributes struct {
	zapcore.Core
	sourceCore zapcore.Core
	attrs      []attribute.KeyValue
	withFields []zap.Field
}

var _ zapcore.Core = coreWithAttributes{}

func (cwa coreWithAttributes) With(fields []zapcore.Field) zapcore.Core {
	cwa.withFields = append(cwa.withFields, fields...)
	cwa.Core = cwa.Core.With(fields)
	return cwa
}

func LoggerWithAttributes(logger *zap.Logger, attrs []attribute.KeyValue) *zap.Logger {
	return logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return coreWithAttributes{
			Core:       c.With([]zap.Field{makeScopeField(attrs)}),
			sourceCore: c,
			attrs:      attrs,
		}
	}))
}

func (cwa coreWithAttributes) DropInjectedAttributes(droppedAttrs ...string) zapcore.Core {
	cwa.attrs = slices.DeleteFunc(slices.Clone(cwa.attrs), func(kv attribute.KeyValue) bool {
		return slices.Contains(droppedAttrs, string(kv.Key))
	})
	cwa.Core = cwa.sourceCore.With([]zap.Field{makeScopeField(cwa.attrs)}).With(cwa.withFields)
	return cwa
}
