// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterexperimental // import "go.opentelemetry.io/collector/exporterexperimental"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
)

// Factory is factory interface for exporters.
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
type Factory interface {
	component.Factory

	// CreateProfilesExporter creates a ProfilesExporter based on this config.
	// If the exporter type does not support tracing or if the config is not valid,
	// an error will be returned instead.
	CreateProfilesExporter(ctx context.Context, set exporter.Settings, cfg component.Config) (Profiles, error)

	// ProfilesExporterStability gets the stability level of the ProfilesExporter.
	ProfilesExporterStability() component.StabilityLevel

	unexportedFactoryFunc()
}

// FactoryOption apply changes to Factory.
type FactoryOption interface {
	// applyExporterFactoryOption applies the option.
	applyExporterFactoryOption(o *factory)
}

var _ FactoryOption = (*factoryOptionFunc)(nil)

// factoryOptionFunc is an ExporterFactoryOption created through a function.
type factoryOptionFunc func(*factory)

func (f factoryOptionFunc) applyExporterFactoryOption(o *factory) {
	f(o)
}

// CreateProfilesFunc is the equivalent of Factory.CreateProfiles.
type CreateProfilesFunc func(context.Context, exporter.Settings, component.Config) (Profiles, error)

// CreateProfilesExporter implements ExporterFactory.CreateProfilesExporter().
func (f CreateProfilesFunc) CreateProfilesExporter(ctx context.Context, set exporter.Settings, cfg component.Config) (Profiles, error) {
	if f == nil {
		return nil, component.ErrDataTypeIsNotSupported
	}
	return f(ctx, set, cfg)
}

type factory struct {
	cfgType component.Type
	component.CreateDefaultConfigFunc
	CreateProfilesFunc
	profilesStabilityLevel component.StabilityLevel
}

func (f *factory) Type() component.Type {
	return f.cfgType
}

func (f *factory) unexportedFactoryFunc() {}

func (f *factory) ProfilesExporterStability() component.StabilityLevel {
	return f.profilesStabilityLevel
}

// WithProfiless overrides the default "error not supported" implementation for CreateProfilesExporter and the default "undefined" stability level.
func WithProfiles(createProfiles CreateProfilesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.profilesStabilityLevel = sl
		o.CreateProfilesFunc = createProfiles
	})
}

// NewFactory returns a Factory.
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	f := &factory{
		cfgType:                 cfgType,
		CreateDefaultConfigFunc: createDefaultConfig,
	}
	for _, opt := range options {
		opt.applyExporterFactoryOption(f)
	}
	return f
}

// MakeFactoryMap takes a list of factories and returns a map with Factory type as keys.
// It returns a non-nil error when there are factories with duplicate type.
func MakeFactoryMap(factories ...Factory) (map[component.Type]Factory, error) {
	fMap := map[component.Type]Factory{}
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return fMap, fmt.Errorf("duplicate exporter factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}

func (b *Builder) Factory(componentType component.Type) component.Factory {
	return b.factories[componentType]
}
