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
	withAttributeSet(attribute.Set) coreWithAttributes
}

func tryWithAttributeSet(c zapcore.Core, attrs attribute.Set) zapcore.Core {
	if cwa, ok := c.(coreWithAttributes); ok {
		return cwa.withAttributeSet(attrs)
	} else {
		return c
	}
}

type consoleCoreWithAttributes struct {
	zapcore.Core
	from zapcore.Core
}

var _ coreWithAttributes = consoleCoreWithAttributes{}

func NewConsoleCoreWithAttributes(c zapcore.Core, attrs attribute.Set) coreWithAttributes {
	var fields []zap.Field
	for _, kv := range attrs.ToSlice() {
		fields = append(fields, zap.String(string(kv.Key), kv.Value.AsString()))
	}
	return consoleCoreWithAttributes{
		Core: c.With(fields),
		from: c,
	}
}

func (ccwa consoleCoreWithAttributes) withAttributeSet(attrs attribute.Set) coreWithAttributes {
	return NewConsoleCoreWithAttributes(ccwa.from, attrs)
}

type otelTeeCoreWithAttributes struct {
	zapcore.Core
	consoleCore zapcore.Core
	lp          log.LoggerProvider
	scopeName   string
	level       zapcore.Level
}

var _ coreWithAttributes = otelTeeCoreWithAttributes{}

func NewOTelTeeCoreWithAttributes(consoleCore zapcore.Core, lp log.LoggerProvider, scopeName string, level zapcore.Level, attrs attribute.Set) otelTeeCoreWithAttributes {
	// TODO: Use otelzap.WithScopeAttributes when it becomes a thing that exists
	lpwa := LoggerProviderWithAttributes(lp, attrs)
	otelCore, err := zapcore.NewIncreaseLevelCore(otelzap.NewCore(
		scopeName,
		otelzap.WithLoggerProvider(lpwa),
	), zap.NewAtomicLevelAt(level))
	if err != nil {
		panic(err)
	}

	return otelTeeCoreWithAttributes{
		Core:        zapcore.NewTee(consoleCore, otelCore),
		consoleCore: consoleCore,
		lp:          lp,
		scopeName:   scopeName,
		level:       level,
	}
}

func (ocwa otelTeeCoreWithAttributes) withAttributeSet(attrs attribute.Set) coreWithAttributes {
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

var _ coreWithAttributes = wrapperCoreWithAttributes{}

func NewWrapperCoreWithAttributes(from zapcore.Core, wrapper func(zapcore.Core) zapcore.Core) coreWithAttributes {
	return wrapperCoreWithAttributes{
		Core:    wrapper(from),
		from:    from,
		wrapper: wrapper,
	}
}

func (wcwa wrapperCoreWithAttributes) withAttributeSet(attrs attribute.Set) coreWithAttributes {
	return NewWrapperCoreWithAttributes(tryWithAttributeSet(wcwa.from, attrs), wcwa.wrapper)
}

func ZapLoggerWithAttributes(logger *zap.Logger, attrs attribute.Set) *zap.Logger {
	return logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return tryWithAttributeSet(c, attrs)
	}))
}
