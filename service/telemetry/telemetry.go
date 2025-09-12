// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"

	otelconf "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/otel/log"
	nooplog "go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/service/telemetry/internal/migration"
)

// NOTE TracesConfig will be removed once opentelemetry-collector-contrib
// has been updated to use otelconftelemetry instead; use at your own risk.
// See https://github.com/open-telemetry/opentelemetry-collector/issues/4970
type TracesConfig = migration.TracesConfigV030

// LoggerSettings holds settings for building logger providers.
type LoggerSettings struct {
	Settings

	// ZapOptions contains options for creating the zap logger.
	ZapOptions []zap.Option
}

// MeterSettings holds settings for building meter providers.
type MeterSettings struct {
	Settings

	// Logger is a zap.Logger that may be used for logging details
	// of the MeterProvider's construction, and by the MeterProvider
	// for logging its internal operations.
	Logger *zap.Logger

	// DefaultViews holds a function that returns default metric
	// views for the given internal telemetry metrics level.
	//
	// The meter provider is expected to use this if no user-provided
	// view configuration is supplied.
	//
	// TODO we should not use otelconf.View directly here, change
	// to something independent of otelconf.
	DefaultViews func(configtelemetry.Level) []otelconf.View
}

// TracerSettings holds settings for building tracer providers.
type TracerSettings struct {
	Settings

	// Logger is a zap.Logger that may be used for logging details
	// of the TracerProvider's construction, and by the TracerProvider
	// for logging its internal operations.
	Logger *zap.Logger
}

// Settings holds common settings for building telemetry providers.
type Settings struct {
	// BuildInfo contains build information about the collector.
	BuildInfo component.BuildInfo
}

// Factory is a factory interface for internal telemetry.
//
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
//
// NOTE This API is experimental and will change soon - use at your own risk.
// See https://github.com/open-telemetry/opentelemetry-collector/issues/4970
type Factory interface {
	// CreateDefaultConfig creates the default configuration for the telemetry.
	CreateDefaultConfig() component.Config

	// CreateResource creates a pcommon.Resource representing the collector.
	// This may be used by components in their internal telemetry.
	CreateResource(context.Context, Settings, component.Config) (pcommon.Resource, error)

	// CreateLogger creates a zap.Logger and LoggerProvider that may be used
	// by components to log their internal operations.
	//
	// NOTE: from the perspective of the Factory implementation, the Logger
	// and the LoggerProvider are independent. However, the service package
	// will arrange for logs written to this logger to be copied to the
	// LoggerProvider.
	CreateLogger(context.Context, LoggerSettings, component.Config) (*zap.Logger, LoggerProvider, error)

	// CreateMeterProvider creates a metric.MeterProvider that may be used
	// by components to record metrics relating to their internal operations.
	CreateMeterProvider(context.Context, MeterSettings, component.Config) (MeterProvider, error)

	// CreateTracerProvider creates a trace.TracerProvider that may be used
	// by components to trace their internal operations.
	CreateTracerProvider(context.Context, TracerSettings, component.Config) (TracerProvider, error)

	// unexportedFactoryFunc is used to prevent external implementations of Factory.
	unexportedFactoryFunc()
}

// LoggerProvider is a log.LoggerProvider that can be shutdown.
type LoggerProvider interface {
	log.LoggerProvider
	Shutdown(context.Context) error
}

// MeterProvider is a metric.MeterProvider that can be shutdown.
type MeterProvider interface {
	metric.MeterProvider
	Shutdown(context.Context) error
}

// TracerProvider is a trace.TracerProvider that can be shutdown.
type TracerProvider interface {
	trace.TracerProvider
	Shutdown(context.Context) error
}

type FactoryOption interface {
	applyOption(*factory)
}

// factoryOptionFunc is an FactoryOption created through a function.
type factoryOptionFunc func(*factory)

func (f factoryOptionFunc) applyOption(o *factory) {
	f(o)
}

type factory struct {
	component.CreateDefaultConfigFunc
	createResourceFunc       CreateResourceFunc
	createLoggerFunc         CreateLoggerFunc
	createMeterProviderFunc  CreateMeterProviderFunc
	createTracerProviderFunc CreateTracerProviderFunc
}

// NewFactory returns a Factory.
func NewFactory(createDefaultConfig component.CreateDefaultConfigFunc, opts ...FactoryOption) Factory {
	f := &factory{
		CreateDefaultConfigFunc: createDefaultConfig,
	}
	for _, opt := range opts {
		opt.applyOption(f)
	}
	return f
}

// WithCreateResource overrides the default CreateResource implementation,
// which creates an empty resource.
func WithCreateResource(createResource CreateResourceFunc) FactoryOption {
	return factoryOptionFunc(func(f *factory) {
		f.createResourceFunc = createResource
	})
}

// CreateResourceFunc is the equivalent of Factory.CreateResource.
type CreateResourceFunc func(context.Context, Settings, component.Config) (pcommon.Resource, error)

// WithCreateLogger overrides the default CreateLogger implementation,
// which creates a noop logger and noop logger provider.
func WithCreateLogger(createLogger CreateLoggerFunc) FactoryOption {
	return factoryOptionFunc(func(f *factory) {
		f.createLoggerFunc = createLogger
	})
}

// WithCreateLogger is the equivalent of Factory.CreateLogger.
type CreateLoggerFunc func(context.Context, LoggerSettings, component.Config) (*zap.Logger, LoggerProvider, error)

// WithCreateMeterProvider overrides the default CreateMeterProvider
func WithCreateMeterProvider(createMeterProvider CreateMeterProviderFunc) FactoryOption {
	return factoryOptionFunc(func(f *factory) {
		f.createMeterProviderFunc = createMeterProvider
	})
}

// CreateMeterProviderFunc is the equivalent of Factory.CreateMeterProvider.
type CreateMeterProviderFunc func(context.Context, MeterSettings, component.Config) (MeterProvider, error)

// WithCreateTracerProvider overrides the default CreateTracerProvider
// implementation, which creates a noop tracer provider.
func WithCreateTracerProvider(createTracerProvider CreateTracerProviderFunc) FactoryOption {
	return factoryOptionFunc(func(f *factory) {
		f.createTracerProviderFunc = createTracerProvider
	})
}

// CreateTracerProviderFunc is the equivalent of Factory.CreateTracerProvider.
type CreateTracerProviderFunc func(context.Context, TracerSettings, component.Config) (TracerProvider, error)

func (*factory) unexportedFactoryFunc() {}

func (f *factory) CreateResource(ctx context.Context, settings Settings, cfg component.Config) (pcommon.Resource, error) {
	if f.createResourceFunc == nil {
		return pcommon.NewResource(), nil
	}
	return f.createResourceFunc(ctx, settings, cfg)
}

func (f *factory) CreateLogger(ctx context.Context, settings LoggerSettings, cfg component.Config) (*zap.Logger, LoggerProvider, error) {
	if f.createLoggerFunc == nil {
		logger := zap.NewNop()
		return logger, noopLoggerProvider{LoggerProvider: nooplog.NewLoggerProvider()}, nil
	}
	return f.createLoggerFunc(ctx, settings, cfg)
}

func (f *factory) CreateMeterProvider(ctx context.Context, settings MeterSettings, cfg component.Config) (MeterProvider, error) {
	if f.createMeterProviderFunc == nil {
		return noopMeterProvider{MeterProvider: noopmetric.NewMeterProvider()}, nil
	}
	return f.createMeterProviderFunc(ctx, settings, cfg)
}

func (f *factory) CreateTracerProvider(ctx context.Context, settings TracerSettings, cfg component.Config) (TracerProvider, error) {
	if f.createTracerProviderFunc == nil {
		return noopTracerProvider{TracerProvider: nooptrace.NewTracerProvider()}, nil
	}
	return f.createTracerProviderFunc(ctx, settings, cfg)
}

type noopLoggerProvider struct {
	nooplog.LoggerProvider
	component.ShutdownFunc
}

type noopMeterProvider struct {
	noopmetric.MeterProvider
	component.ShutdownFunc
}

type noopTracerProvider struct {
	nooptrace.TracerProvider
	component.ShutdownFunc
}
