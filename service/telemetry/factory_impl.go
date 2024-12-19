// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"

	"go.opentelemetry.io/otel/log"
	lognoop "go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
)

// factoryOption apply changes to Factory.
type factoryOption interface {
	// applyTelemetryFactoryOption applies the option.
	applyTelemetryFactoryOption(o *factory)
}

var _ factoryOption = (*factoryOptionFunc)(nil)

// factoryOptionFunc is an factoryOption created through a function.
type factoryOptionFunc func(*factory)

func (f factoryOptionFunc) applyTelemetryFactoryOption(o *factory) {
	f(o)
}

var _ Factory = (*factory)(nil)

// Factory is the implementation of Factory.
type factory struct {
	createDefaultConfig component.CreateDefaultConfigFunc
	createLoggerFunc
	createTracerProviderFunc
	createMeterProviderFunc
}

func (f *factory) CreateDefaultConfig() component.Config {
	return f.createDefaultConfig()
}

// createLoggerFunc is the equivalent of Factory.CreateLogger.
type createLoggerFunc func(context.Context, Settings, component.Config) (*zap.Logger, log.LoggerProvider, error)

// withLogger overrides the default no-op logger.
func withLogger(createLogger createLoggerFunc) factoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.createLoggerFunc = createLogger
	})
}

func (f *factory) CreateLogger(ctx context.Context, set Settings, cfg component.Config) (*zap.Logger, log.LoggerProvider, error) {
	if f.createLoggerFunc == nil {
		return zap.NewNop(), lognoop.NewLoggerProvider(), nil
	}
	return f.createLoggerFunc(ctx, set, cfg)
}

// createTracerProviderFunc is the equivalent of Factory.CreateTracerProvider.
type createTracerProviderFunc func(context.Context, Settings, component.Config) (trace.TracerProvider, error)

// withTracerProvider overrides the default no-op tracer provider.
func withTracerProvider(createTracerProvider createTracerProviderFunc) factoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.createTracerProviderFunc = createTracerProvider
	})
}

func (f *factory) CreateTracerProvider(ctx context.Context, set Settings, cfg component.Config) (trace.TracerProvider, error) {
	if f.createTracerProviderFunc == nil {
		return tracenoop.NewTracerProvider(), nil
	}
	return f.createTracerProviderFunc(ctx, set, cfg)
}

// createMeterProviderFunc is the equivalent of Factory.CreateMeterProvider.
type createMeterProviderFunc func(context.Context, Settings, component.Config) (metric.MeterProvider, error)

// withMeterProvider overrides the default no-op meter provider.
func withMeterProvider(createMeterProvider createMeterProviderFunc) factoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.createMeterProviderFunc = createMeterProvider
	})
}

func (f *factory) CreateMeterProvider(ctx context.Context, set Settings, cfg component.Config) (metric.MeterProvider, error) {
	if f.createMeterProviderFunc == nil {
		return metricnoop.NewMeterProvider(), nil
	}
	return f.createMeterProviderFunc(ctx, set, cfg)
}

func (f *factory) unexportedFactoryFunc() {}

// newFactory returns a new Factory.
func newFactory(createDefaultConfig component.CreateDefaultConfigFunc, options ...factoryOption) Factory {
	f := &factory{
		createDefaultConfig: createDefaultConfig,
	}
	for _, op := range options {
		op.applyTelemetryFactoryOption(f)
	}
	return f
}
