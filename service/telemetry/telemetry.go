// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"

	"go.opentelemetry.io/otel/log"
	nooplog "go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/service/telemetry/internal/migration"
)

// NOTE TracesConfig will be removed once opentelemetry-collector-contrib
// has been updated to use otelconftelemetry instead; use at your own risk.
// See https://github.com/open-telemetry/opentelemetry-collector/issues/4970
type TracesConfig = migration.TracesConfigV030

// Providers is an interface for internal telemetry providers.
//
// NOTE this interface is experimental and may change in the future.
type Providers interface {
	// Shutdown gracefully shuts down the telemetry providers.
	Shutdown(context.Context) error

	// Resource returns a pcommon.Resource representing the collector.
	// This may be used by components in their internal telemetry.
	Resource() pcommon.Resource

	// Logger returns a zap.Logger that may be used by components to
	// log their internal operations.
	//
	// NOTE: from the perspective of the Providers implementation,
	// this Logger and the LoggerProvider are independent. However,
	// the service package will arrange for logs written to this
	// logger to be copied to the LoggerProvider. The effective
	// level of the logger will be the lower of the Logger's and
	// the LoggerProvider's levels.
	Logger() *zap.Logger

	// LoggerProvider returns a log.LoggerProvider that may be used
	// for components to log their internal operations.
	LoggerProvider() log.LoggerProvider

	// MeterProvider returns a metric.MeterProvider that may be used
	// by components to record metrics relating to their internal
	// operations.
	MeterProvider() metric.MeterProvider

	// TracerProvider returns a trace.TracerProvider that may be used
	// by components to trace their internal operations.
	TracerProvider() trace.TracerProvider
}

// Settings holds configuration for building Providers.
type Settings struct {
	// BuildInfo contains build information about the collector.
	BuildInfo component.BuildInfo

	// ZapOptions contains options for creating the zap logger.
	ZapOptions []zap.Option
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

	// CreateProviders creates the logger, meter, and tracer providers for
	// the collector's internal telemetry.
	CreateProviders(context.Context, Settings, component.Config) (Providers, error)

	// unexportedFactoryFunc is used to prevent external implementations of Factory.
	unexportedFactoryFunc()
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
	createProvidersFunc CreateProvidersFunc
}

// NewFactory returns a Factory.
//
// If createProviders is nil, then the returned Factory's CreateProviders
// method will return a Providers with noop telemetry providers.
func NewFactory(
	createDefaultConfig component.CreateDefaultConfigFunc,
	createProviders CreateProvidersFunc,
	opts ...FactoryOption,
) Factory {
	f := &factory{
		CreateDefaultConfigFunc: createDefaultConfig,
		createProvidersFunc:     createProviders,
	}
	for _, opt := range opts {
		opt.applyOption(f)
	}
	return f
}

// CreateProvidersFunc is the equivalent of Factory.CreateProviders.
type CreateProvidersFunc func(context.Context, Settings, component.Config) (Providers, error)

func (*factory) unexportedFactoryFunc() {}

func (f *factory) CreateProviders(ctx context.Context, settings Settings, cfg component.Config) (Providers, error) {
	if f.createProvidersFunc == nil {
		return nopProviders{}, nil
	}
	return f.createProvidersFunc(ctx, settings, cfg)
}

type nopProviders struct{}

func (nopProviders) Shutdown(context.Context) error {
	return nil
}

func (nopProviders) Resource() pcommon.Resource {
	return pcommon.NewResource()
}

func (nopProviders) Logger() *zap.Logger {
	return zap.NewNop()
}

func (nopProviders) LoggerProvider() log.LoggerProvider {
	return nooplog.NewLoggerProvider()
}

func (nopProviders) MeterProvider() metric.MeterProvider {
	return noopmetric.NewMeterProvider()
}

func (nopProviders) TracerProvider() trace.TracerProvider {
	return nooptrace.NewTracerProvider()
}
