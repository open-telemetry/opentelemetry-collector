// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"

	lognoop "go.opentelemetry.io/otel/log/noop"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
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
	createResourceFunc
	createProvidersFunc
}

func (f *factory) CreateDefaultConfig() component.Config {
	return f.createDefaultConfig()
}

type createResourceFunc func(context.Context, Settings, component.Config) (pcommon.Resource, error)

func withResource(createResource createResourceFunc) factoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.createResourceFunc = createResource
	})
}

func (f *factory) CreateResource(ctx context.Context, set Settings, cfg component.Config) (pcommon.Resource, error) {
	if f.createResourceFunc == nil {
		return pcommon.NewResource(), nil
	}
	return f.createResourceFunc(ctx, set, cfg)
}

// createProvidersFunc is the equivalent of Factory.CreateProviders.
type createProvidersFunc func(context.Context, Settings, component.Config) (Providers, error)

// withProviders overrides the default no-op telemetry providers.
func withProviders(createProviders createProvidersFunc) factoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.createProvidersFunc = createProviders
	})
}

func (f *factory) CreateProviders(ctx context.Context, set Settings, cfg component.Config) (Providers, error) {
	if f.createProvidersFunc == nil {
		return noopProviders(), nil
	}
	return f.createProvidersFunc(ctx, set, cfg)
}

func noopProviders() Providers {
	return Providers{
		Logger:         zap.NewNop(),
		LoggerProvider: lognoop.NewLoggerProvider(),
		MeterProvider:  metricnoop.NewMeterProvider(),
		TracerProvider: tracenoop.NewTracerProvider(),
	}
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
