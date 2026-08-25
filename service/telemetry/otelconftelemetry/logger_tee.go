// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"slices"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/log"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/service/internal/componentattribute"
)

type zapCoreProvider struct {
	sourceCore zapcore.Core
	lp         log.LoggerProvider
	scopeName  string
}

func (zcp *zapCoreProvider) newCore() zapCore {
	return zapCore{
		sourceCore: zcp.sourceCore,
		otelCore: otelzap.NewCore(
			zcp.scopeName,
			otelzap.WithLoggerProvider(zcp.lp),
		),
		provider: zcp,
	}
}

// This struct wraps the original Zap core in order to copy logs to a [log.LoggerProvider] using [otelzap].
// For the copied logs, component attributes are injected as instrumentation scope attributes.
//
// Note that we intentionally do not use zapcore.NewTee here, because it will simply duplicate all log entries
// to each core. The provided Zap core may have sampling or a minimum log level applied to it, so in order to
// maintain consistency, we need to ensure that only the logs accepted by the provided core are copied to the
// log.LoggerProvider.
type zapCore struct {
	sourceCore zapcore.Core // regular Zap core (logs to stderr)
	otelCore   zapcore.Core // otelzap core (forwards to OTel Logger)
	provider   *zapCoreProvider
	withFields []zap.Field // additional fields injected by the user using .With
}

var _ zapcore.Core = zapCore{}

func (zc zapCore) With(fields []zapcore.Field) zapcore.Core {
	zc.sourceCore = zc.sourceCore.With(fields)
	fields = slices.DeleteFunc(fields, func(field zapcore.Field) bool {
		scope, ok := componentattribute.ExtractLogScopeAttributes(field)
		if !ok {
			return false
		}
		// Set scope attributes
		zc.otelCore = otelzap.NewCore(
			zc.provider.scopeName,
			otelzap.WithLoggerProvider(zc.provider.lp),
			otelzap.WithAttributes(scope...),
		).With(zc.withFields)
		return true
	})
	zc.otelCore = zc.otelCore.With(fields)
	zc.withFields = append(slices.Clone(zc.withFields), fields...)
	return zc
}

func (zc zapCore) Enabled(level zapcore.Level) bool {
	return zc.sourceCore.Enabled(level)
}

func (zc zapCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	ce = zc.sourceCore.Check(entry, ce)
	if ce != nil {
		// Only log to the otelzap core if the source core accepted the log entry.
		ce = ce.AddCore(entry, zc.otelCore)
	}
	return ce
}

// This function should never be called, since only the inner cores add themselves to the CheckedEntry.
// But like zapcore.multiCore, we still implement it for compatibility with non-conforming users.
func (zc zapCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	return multierr.Append(
		zc.sourceCore.Write(entry, fields),
		zc.otelCore.Write(entry, fields),
	)
}

func (zc zapCore) Sync() error {
	return multierr.Append(
		zc.sourceCore.Sync(),
		zc.otelCore.Sync(),
	)
}
