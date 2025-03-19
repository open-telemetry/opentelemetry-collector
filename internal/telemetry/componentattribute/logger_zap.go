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

type coreWithAttributes interface {
	zapcore.Core
	withAttributeSet(attribute.Set) zapcore.Core
}

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
