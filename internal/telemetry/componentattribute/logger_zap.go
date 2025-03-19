// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute // import "go.opentelemetry.io/collector/internal/telemetry/componentattribute"

import (
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Interface for Zap cores that support setting and resetting a set of component attributes.
type coreWithAttributes interface {
	zapcore.Core
	withAttributeSet(attribute.Set) zapcore.Core
}

// Tries setting the attribute set for a Zap core.
// Does nothing if the core does not support setting component attributes.
func tryWithAttributeSet(c zapcore.Core, attrs attribute.Set) zapcore.Core {
	if cwa, ok := c.(coreWithAttributes); ok {
		return cwa.withAttributeSet(attrs)
	}
	return c
}

type consoleCoreWithAttributes struct {
	zapcore.Core
	from zapcore.Core
}

var _ coreWithAttributes = (*consoleCoreWithAttributes)(nil)

// Wraps a Zap core to support adding component attributes as Zap fields.
// Used for console output.
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

// Creates a Zap core which forwards its output to both a provided Zap `consoleCore` and to a `LoggerProvider`.
// Data sent through OTel will have component attributes set as instrumentation scope attributes.
func NewOTelTeeCoreWithAttributes(consoleCore zapcore.Core, lp log.LoggerProvider, scopeName string, level zapcore.Level, attrs attribute.Set) zapcore.Core {
	// TODO: Use `otelzap.WithAttributes` and remove `LoggerProviderWithAttributes`
	// once https://github.com/open-telemetry/opentelemetry-go-contrib/issues/6954 is implemented.
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
	return NewOTelTeeCoreWithAttributes(
		tryWithAttributeSet(ocwa.consoleCore, attrs),
		ocwa.lp, ocwa.scopeName, ocwa.level,
		attrs,
	)
}

type wrapperCoreWithAttributes struct {
	zapcore.Core
	from    zapcore.Core
	wrapper func(zapcore.Core) zapcore.Core
}

var _ coreWithAttributes = (*wrapperCoreWithAttributes)(nil)

// Applies a wrapper function to a Zap core similar to `logger.WithOption(zap.WrapCore(...))`,
// but allows component attribute changes to be forwarded to the underlying core.
// The wrapper function will be run again when the underlying core is recreated.
func NewWrapperCoreWithAttributes(from zapcore.Core, wrapper func(zapcore.Core) zapcore.Core) zapcore.Core {
	return &wrapperCoreWithAttributes{
		Core:    wrapper(from),
		from:    from,
		wrapper: wrapper,
	}
}

func (wcwa *wrapperCoreWithAttributes) withAttributeSet(attrs attribute.Set) zapcore.Core {
	return NewWrapperCoreWithAttributes(tryWithAttributeSet(wcwa.from, attrs), wcwa.wrapper)
}

func ZapLoggerWithAttributes(logger *zap.Logger, attrs attribute.Set) *zap.Logger {
	return logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return tryWithAttributeSet(c, attrs)
	}))
}
