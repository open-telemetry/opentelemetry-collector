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

type wrapCoreFunc = func(zapcore.Core, attribute.Set) zapcore.Core

type serviceCore struct {
	zapcore.Core
	lp        log.LoggerProvider
	scopeName string
	wrapCore  wrapCoreFunc
}

func NewServiceZapCore(lp log.LoggerProvider, scopeName string, wrapCore wrapCoreFunc, attrs attribute.Set) zapcore.Core {
	// TODO: Use otelzap.WithScopeAttributes when it becomes a thing that exists
	lpwa := LoggerProviderWithAttributes(lp, attrs)
	var core zapcore.Core = otelzap.NewCore(
		scopeName,
		otelzap.WithLoggerProvider(lpwa),
	)
	if wrapCore != nil {
		core = wrapCore(core, attrs)
	}
	return serviceCore{
		Core:      core,
		lp:        lp,
		scopeName: scopeName,
		wrapCore:  wrapCore,
	}
}

func ZapLoggerWithAttributes(logger *zap.Logger, attrs attribute.Set) *zap.Logger {
	return logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		sc, ok := c.(serviceCore)
		if !ok {
			// Logger does not use OTel under the hood, cannot change Logger parameters
			return c
		}
		return NewServiceZapCore(sc.lp, sc.scopeName, sc.wrapCore, attrs)
	}))
}
