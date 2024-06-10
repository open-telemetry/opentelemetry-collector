// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/service/telemetry/internal"

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
)

// CreateSettings holds configuration for building Telemetry.
type CreateSettings struct {
	BuildInfo         component.BuildInfo
	AsyncErrorChannel chan error
	ZapOptions        []zap.Option
}

// Factory is factory interface for telemetry.
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
type Factory interface {
	// CreateDefaultConfig creates the default configuration for the telemetry.
	// TODO: Should we just inherit from component.Factory?
	CreateDefaultConfig() component.Config

	// CreateLogger creates a logger.
	CreateLogger(ctx context.Context, set CreateSettings, cfg component.Config) (*zap.Logger, error)

	// CreateTracerProvider creates a TracerProvider.
	CreateTracerProvider(ctx context.Context, set CreateSettings, cfg component.Config) (trace.TracerProvider, error)

	// TODO: Add CreateMeterProvider.

	// unexportedFactoryFunc is used to prevent external implementations of Factory.
	unexportedFactoryFunc()
}

// FactoryOption apply changes to Factory.
type FactoryOption interface {
	// applyTelemetryFactoryOption applies the option.
	applyTelemetryFactoryOption(o *factory)
}

var _ FactoryOption = (*factoryOptionFunc)(nil)

// factoryOptionFunc is an FactoryOption created through a function.
type factoryOptionFunc func(*factory)

func (f factoryOptionFunc) applyTelemetryFactoryOption(o *factory) {
	f(o)
}

var _ Factory = (*factory)(nil)

// factory is the implementation of Factory.
type factory struct {
	createDefaultConfig component.CreateDefaultConfigFunc
	CreateLoggerFunc
	CreateTracerProviderFunc
}

func (f *factory) CreateDefaultConfig() component.Config {
	return f.createDefaultConfig()
}

// CreateLoggerFunc is the equivalent of Factory.CreateLogger.
type CreateLoggerFunc func(context.Context, CreateSettings, component.Config) (*zap.Logger, error)

// WithLogger overrides the default no-op logger.
func WithLogger(createLogger CreateLoggerFunc) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.CreateLoggerFunc = createLogger
	})
}

func (f *factory) CreateLogger(ctx context.Context, set CreateSettings, cfg component.Config) (*zap.Logger, error) {
	if f.CreateLoggerFunc == nil {
		return zap.NewNop(), nil
	}
	return f.CreateLoggerFunc(ctx, set, cfg)
}

// CreateTracerProviderFunc is the equivalent of Factory.CreateTracerProvider.
type CreateTracerProviderFunc func(context.Context, CreateSettings, component.Config) (trace.TracerProvider, error)

// WithTracerProvider overrides the default no-op tracer provider.
func WithTracerProvider(createTracerProvider CreateTracerProviderFunc) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.CreateTracerProviderFunc = createTracerProvider
	})
}

func (f *factory) CreateTracerProvider(ctx context.Context, set CreateSettings, cfg component.Config) (trace.TracerProvider, error) {
	if f.CreateTracerProviderFunc == nil {
		return tracenoop.NewTracerProvider(), nil
	}
	return f.CreateTracerProviderFunc(ctx, set, cfg)
}

func (f *factory) unexportedFactoryFunc() {}

// NewFactory returns a new Factory.
func NewFactory(createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	f := &factory{
		createDefaultConfig: createDefaultConfig,
	}
	for _, op := range options {
		op.applyTelemetryFactoryOption(f)
	}
	return f
}
