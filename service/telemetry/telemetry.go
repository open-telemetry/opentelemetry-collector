// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"

	otelconf "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// LoggerSettings holds settings for building logger providers.
type LoggerSettings struct {
	Settings

	// ZapOptions contains options for creating the zap logger.
	//
	// Deprecated [v0.142.0]: use BuildZapLogger instead.
	// This field will be removed in the future, and options
	// must be injected through BuildZapLogger.
	ZapOptions []zap.Option

	// BuildZapLogger holds a function for building a *zap.Logger
	// from a zap.Config and options.
	//
	// If BuildZapLogger is nil, zap.Config.Build should be used.
	// NOTE: in the future this field will be required.
	BuildZapLogger func(zap.Config, ...zap.Option) (*zap.Logger, error)
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

	// Resource is the telemetry resource that should be used by all telemetry providers.
	Resource *pcommon.Resource
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

	// CreateLogger creates a zap.Logger that may be used by components to
	// log their internal operations, along with a function that must be called
	// when the service is shutting down.
	CreateLogger(context.Context, LoggerSettings, component.Config) (*zap.Logger, component.ShutdownFunc, error)

	// CreateMeterProvider creates a metric.MeterProvider that may be used
	// by components to record metrics relating to their internal operations.
	CreateMeterProvider(context.Context, MeterSettings, component.Config) (MeterProvider, error)

	// CreateTracerProvider creates a trace.TracerProvider that may be used
	// by components to trace their internal operations.
	//
	// If the returned provider is a wrapper, consider implementing
	// the `Unwrap() trace.TracerProvider` method to grant components access to the underlying SDK.
	CreateTracerProvider(context.Context, TracerSettings, component.Config) (TracerProvider, error)

	// unexportedFactoryFunc is used to prevent external implementations of Factory.
	unexportedFactoryFunc()
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
type CreateLoggerFunc func(context.Context, LoggerSettings, component.Config) (*zap.Logger, component.ShutdownFunc, error)

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

func (f *factory) CreateLogger(ctx context.Context, settings LoggerSettings, cfg component.Config) (*zap.Logger, component.ShutdownFunc, error) {
	if f.createLoggerFunc == nil {
		logger := zap.NewNop()
		nopShutdown := component.ShutdownFunc(nil)
		return logger, nopShutdown, nil
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

type noopMeterProvider struct {
	noopmetric.MeterProvider
	component.ShutdownFunc
}

type noopTracerProvider struct {
	nooptrace.TracerProvider
	component.ShutdownFunc
}
