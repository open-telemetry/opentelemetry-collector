// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component // import "go.opentelemetry.io/collector/component"

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// TelemetrySettings provides components with APIs to report telemetry.
type TelemetrySettings struct {
	// Logger that the factory can use during creation and can pass to the created
	// component to be used later as well.
	Logger *zap.Logger

	// TracerProvider that the factory can pass to other instrumented third-party libraries.
	TracerProvider trace.TracerProvider

	// MeterProvider that the factory can pass to other instrumented third-party libraries.
	MeterProvider metric.MeterProvider

	// Resource contains the resource attributes for the collector's telemetry.
	Resource pcommon.Resource
}

func (ts *TelemetrySettings) LoggerWithout(fields ...string) *zap.Logger {
	type coreWithout interface {
		Without(fields ...string) zapcore.Core
	}
	if _, ok := ts.Logger.Core().(coreWithout); !ok {
		return ts.Logger
	}
	return ts.Logger.WithOptions(
		zap.WrapCore(func(from zapcore.Core) zapcore.Core {
			return from.(coreWithout).Without(fields...)
		}),
	)
}
