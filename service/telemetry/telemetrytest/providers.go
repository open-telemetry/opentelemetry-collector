// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetrytest // import "go.opentelemetry.io/collector/service/telemetry/telemetrytest"

import (
	"context"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/service/telemetry"
)

// WithResource returns a telemetry.FactoryOption that configures the
// factory's CreateResource method to return res.
func WithResource(res pcommon.Resource) telemetry.FactoryOption {
	return telemetry.WithCreateResource(
		func(context.Context, telemetry.Settings, component.Config) (pcommon.Resource, error) {
			return res, nil
		},
	)
}

// WithLogger returns a telemetry.FactoryOption that configures the
// factory's CreateLogger method to return logger and shutdown func.
func WithLogger(logger *zap.Logger, shutdownFunc component.ShutdownFunc) telemetry.FactoryOption {
	return telemetry.WithCreateLogger(
		func(context.Context, telemetry.LoggerSettings, component.Config) (
			*zap.Logger, component.ShutdownFunc, error,
		) {
			return logger, shutdownFunc, nil
		},
	)
}

// WithMeterProvider returns a telemetry.FactoryOption that configures the
// factory's CreateMeterProvider method to return provider. If provider does
// not implement the Shutdown method, it will be wrapped with
// ShutdownMeterProvider with a no-op shutdown func.
func WithMeterProvider(provider metric.MeterProvider) telemetry.FactoryOption {
	return telemetry.WithCreateMeterProvider(
		func(context.Context, telemetry.MeterSettings, component.Config) (
			telemetry.MeterProvider, error,
		) {
			withShutdown, ok := provider.(telemetry.MeterProvider)
			if !ok {
				withShutdown = ShutdownMeterProvider{MeterProvider: provider}
			}
			return withShutdown, nil
		},
	)
}

// WithTracerProvider returns a telemetry.FactoryOption that configures the
// factory's CreateTracerProvider method to return provider. If provider does
// not implement the Shutdown method, it will be wrapped with
// ShutdownTracerProvider with a no-op shutdown func.
func WithTracerProvider(provider trace.TracerProvider) telemetry.FactoryOption {
	return telemetry.WithCreateTracerProvider(
		func(context.Context, telemetry.TracerSettings, component.Config) (
			telemetry.TracerProvider, error,
		) {
			withShutdown, ok := provider.(telemetry.TracerProvider)
			if !ok {
				withShutdown = ShutdownTracerProvider{TracerProvider: provider}
			}
			return withShutdown, nil
		},
	)
}

type ShutdownMeterProvider struct {
	metric.MeterProvider
	component.ShutdownFunc
}

type ShutdownTracerProvider struct {
	trace.TracerProvider
	component.ShutdownFunc
}
