// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute // import "go.opentelemetry.io/collector/internal/telemetry/componentattribute"

import (
	"reflect"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Interface for Zap cores that support setting and resetting a set of component attributes.
//
// There are three wrappers that implement this interface:
//
//   - [NewConsoleCoreWithAttributes] injects component attributes as Zap fields.
//
//     This is used for the Collector's console output.
//
//   - [NewOTelTeeCoreWithAttributes] copies logs to a [log.LoggerProvider] using [otelzap]. For the
//     copied logs, component attributes are injected as instrumentation scope attributes.
//
//     This is used when service::telemetry::logs::processors is configured.
//
//   - [NewWrapperCoreWithAttributes] applies a wrapper function to a core, similar to
//     [zap.WrapCore]. It allows setting component attributes on the inner core and reapplying the
//     wrapper function when needed.
//
//     This is used when adding [zapcore.NewSamplerWithOptions] to our logger stack.
type coreWithAttributes interface {
	zapcore.Core
	withAttributeSet(attribute.Set) zapcore.Core
}

// Tries setting the component attribute set for a Zap core.
//
// Does nothing if the core does not implement [coreWithAttributes].
func tryWithAttributeSet(c zapcore.Core, attrs attribute.Set) zapcore.Core {
	if cwa, ok := c.(coreWithAttributes); ok {
		return cwa.withAttributeSet(attrs)
	}
	zap.New(c).Debug("Logger core does not support injecting component attributes")
	return c
}

type consoleCoreWithAttributes struct {
	zapcore.Core
	from zapcore.Core
}

var _ coreWithAttributes = (*consoleCoreWithAttributes)(nil)

// NewConsoleCoreWithAttributes wraps a Zap core in order to inject component attributes as Zap fields.
//
// This is used for the Collector's console output.
func NewConsoleCoreWithAttributes(c zapcore.Core, attrs attribute.Set) zapcore.Core {
	var fields []zap.Field
	for _, kv := range attrs.ToSlice() {
		fields = append(fields, zap.String(string(kv.Key), kv.Value.AsString()))
	}
	return &consoleCoreWithAttributes{
		Core: c.With(fields),
		from: c,
	}
}

func (ccwa *consoleCoreWithAttributes) withAttributeSet(attrs attribute.Set) zapcore.Core {
	return NewConsoleCoreWithAttributes(ccwa.from, attrs)
}

type otelTeeCoreWithAttributes struct {
	zapcore.Core
	consoleCore zapcore.Core
	lp          log.LoggerProvider
	scopeName   string
	level       zapcore.Level
}

var _ coreWithAttributes = (*otelTeeCoreWithAttributes)(nil)

// NewOTelTeeCoreWithAttributes wraps a Zap core in order to copy logs to a [log.LoggerProvider] using [otelzap]. For the copied
// logs, component attributes are injected as instrumentation scope attributes.
//
// This is used when service::telemetry::logs::processors is configured.
func NewOTelTeeCoreWithAttributes(consoleCore zapcore.Core, lp log.LoggerProvider, scopeName string, level zapcore.Level, attrs attribute.Set) zapcore.Core {
	// TODO: Use `otelzap.WithAttributes` and remove `LoggerProviderWithAttributes`
	// once we've upgraded to otelzap v0.11.0.
	lpwa := LoggerProviderWithAttributes(lp, attrs)
	otelCore, err := zapcore.NewIncreaseLevelCore(otelzap.NewCore(
		scopeName,
		otelzap.WithLoggerProvider(lpwa),
	), zap.NewAtomicLevelAt(level))
	if err != nil {
		panic(err)
	}

	return &otelTeeCoreWithAttributes{
		Core:        zapcore.NewTee(consoleCore, otelCore),
		consoleCore: consoleCore,
		lp:          lp,
		scopeName:   scopeName,
		level:       level,
	}
}

func (ocwa *otelTeeCoreWithAttributes) withAttributeSet(attrs attribute.Set) zapcore.Core {
	return NewOTelTeeCoreWithAttributes(tryWithAttributeSet(ocwa.consoleCore, attrs), ocwa.lp, ocwa.scopeName, ocwa.level, attrs)
}

type samplerCoreWithAttributes struct {
	zapcore.Core
	from zapcore.Core
}

var _ coreWithAttributes = (*samplerCoreWithAttributes)(nil)

func NewSamplerCoreWithAttributes(inner zapcore.Core, tick time.Duration, first int, thereafter int) zapcore.Core {
	return &samplerCoreWithAttributes{
		Core: zapcore.NewSamplerWithOptions(inner, tick, first, thereafter),
		from: inner,
	}
}

func (ssc *samplerCoreWithAttributes) withAttributeSet(attrs attribute.Set) zapcore.Core {
	newInner := tryWithAttributeSet(ssc.from, attrs)

	// See the code for zap sampler cores:
	// https://github.com/uber-go/zap/blob/fcf8ee58669e358bbd6460bef5c2ee7a53c0803a/zapcore/sampler.go#L168
	// The `counts` allocation is very large, but can be reused across samplers (see `With` method).
	// However, there is no method to change the inner core, so we call upon `reflect` magic to do it.
	sampler1 := ssc.Core
	val1 := reflect.ValueOf(sampler1).Elem()
	val2 := reflect.New(val1.Type())
	val2.Elem().Set(val1)
	val2.Elem().FieldByName("Core").Set(reflect.ValueOf(newInner))
	sampler2 := val2.Interface().(zapcore.Core)

	return samplerCoreWithAttributes{
		Core: sampler2,
		from: newInner,
	}
}

// ZapLoggerWithAttributes creates a Zap Logger with a new set of injected component attributes.
func ZapLoggerWithAttributes(logger *zap.Logger, attrs attribute.Set) *zap.Logger {
	return logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return tryWithAttributeSet(c, attrs)
	}))
}
