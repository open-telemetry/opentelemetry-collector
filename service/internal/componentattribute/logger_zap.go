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
type scopeAttributesField struct {
	fields []zapcore.Field
	attrs  []attribute.KeyValue
}

var _ zapcore.ObjectMarshaler = scopeAttributesField{}

func (saf scopeAttributesField) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for _, field := range saf.fields {
		field.AddTo(enc)
	}
	return nil
}

func makeScopeField(attrs []attribute.KeyValue) zap.Field {
	return zap.Inline(scopeAttributesField{
		fields: telemetry.ToZapFields(attrs),
		attrs:  attrs,
	})
}

func ExtractLogScopeAttributes(field zap.Field) ([]attribute.KeyValue, bool) {
	if field.Type != zapcore.InlineMarshalerType {
		return nil, false
	}
	saf, ok := field.Interface.(scopeAttributesField)
	if !ok {
		return nil, false
	}
	return saf.attrs, true
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
	cwa.Core = cwa.sourceCore.With(append([]zap.Field{makeScopeField(cwa.attrs)}, cwa.withFields...))
	return cwa
}
