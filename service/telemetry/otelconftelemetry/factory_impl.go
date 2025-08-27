// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/telemetry"
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
	createProvidersFunc
}

func (f *factory) CreateDefaultConfig() component.Config {
	return f.createDefaultConfig()
}

// createProvidersFunc is the equivalent of Factory.CreateProviders.
type createProvidersFunc func(context.Context, telemetry.Settings, component.Config) (telemetry.Providers, error)

func (f *factory) CreateProviders(ctx context.Context, set telemetry.Settings, cfg component.Config) (telemetry.Providers, error) {
	return f.createProvidersFunc(ctx, set, cfg)
}

func (f *factory) unexportedFactoryFunc() {}

// newFactory returns a new Factory.
func newFactory(createDefaultConfig component.CreateDefaultConfigFunc, createProviders createProvidersFunc, options ...factoryOption) Factory {
	f := &factory{
		createDefaultConfig: createDefaultConfig,
		createProvidersFunc: createProviders,
	}
	for _, op := range options {
		op.applyTelemetryFactoryOption(f)
	}
	return f
}
