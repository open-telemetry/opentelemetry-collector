// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute // import "go.opentelemetry.io/collector/service/internal/componentattribute"

import (
	"slices"

	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
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

// levelOverrideCore wraps a zapcore.Core to override its level so that
// ALL messages at or above the override level are passed through,
// regardless of the underlying core's configured level.
type levelOverrideCore struct {
	zapcore.Core
	overrideLevel zapcore.Level
}

func (loc levelOverrideCore) Enabled(l zapcore.Level) bool {
	// We only check against our override level, NOT the underlying core's level.
	// This is intentional: the override is meant to take full control.
	return l >= loc.overrideLevel
}

func (loc levelOverrideCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	// Use our own Enabled, then delegate Write to underlying core.
	if loc.Enabled(e.Level) {
		return ce.AddCore(e, loc.Core)
	}
	return ce
}

// LoggerWithLevel returns a new *zap.Logger that logs at the given level
// or above, regardless of the underlying logger's configured level.
// This is used for per-component log level overrides.
func LoggerWithLevel(logger *zap.Logger, level zapcore.Level) *zap.Logger {
	return logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return levelOverrideCore{Core: c, overrideLevel: level}
	}))
}

// ApplyLogLevelOverride returns a copy of TelemetrySettings with the logger's
// log level overridden if a per-component override exists for the given component ID.
// If no override exists, the original TelemetrySettings is returned unchanged.
func ApplyLogLevelOverride(tel component.TelemetrySettings, overrides map[component.ID]*zapcore.Level, id component.ID) component.TelemetrySettings {
	if overrides == nil {
		return tel
	}
	level, ok := overrides[id]
	if !ok || level == nil {
		return tel
	}
	tel.Logger = LoggerWithLevel(tel.Logger, *level)
	return tel
}
