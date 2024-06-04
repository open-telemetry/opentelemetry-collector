// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorexperimental // import "go.opentelemetry.io/collector/processorexperimental"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumerexperimental"
	"go.opentelemetry.io/collector/processor"
)

// Factory is Factory interface for processors.
// This interface cannot be directly implemented. Implementations must
// use the NewProcessorFactory to implement it.
type Factory interface {
	component.Factory

	// CreateProfilesProcessor creates a ProfilesProcessor based on this config.
	// If the processor type does not support tracing or if the config is not valid,
	// an error will be returned instead.
	CreateProfilesProcessor(ctx context.Context, set processor.CreateSettings, cfg component.Config, nextConsumer consumerexperimental.Profiles) (Profiles, error)

	// ProfilesProcessorStability gets the stability level of the ProfilesProcessor.
	ProfilesProcessorStability() component.StabilityLevel

	unexportedFactoryFunc()
}

// FactoryOption apply changes to Options.
type FactoryOption interface {
	// applyProcessorFactoryOption applies the option.
	applyProcessorFactoryOption(o *factory)
}

var _ FactoryOption = (*factoryOptionFunc)(nil)

// factoryOptionFunc is a FactoryOption created through a function.
type factoryOptionFunc func(*factory)

func (f factoryOptionFunc) applyProcessorFactoryOption(o *factory) {
	f(o)
}

// CreateProfilesFunc is the equivalent of Factory.CreateProfiles().
type CreateProfilesFunc func(context.Context, processor.CreateSettings, component.Config, consumerexperimental.Profiles) (Profiles, error)

// CreateProfilesProcessor implements Factory.CreateProfilesProcessor().
func (f CreateProfilesFunc) CreateProfilesProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumerexperimental.Profiles) (Profiles, error) {
	if f == nil {
		return nil, component.ErrDataTypeIsNotSupported
	}
	return f(ctx, set, cfg, nextConsumer)
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

func (f factory) ProfilesProcessorStability() component.StabilityLevel {
	return f.profilesStabilityLevel
}

// WithProfiless overrides the default "error not supported" implementation for CreateProfiles and the default "undefined" stability level.
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
		opt.applyProcessorFactoryOption(f)
	}
	return f
}

// MakeFactoryMap takes a list of factories and returns a map with Factory type as keys.
// It returns a non-nil error when there are factories with duplicate type.
func MakeFactoryMap(factories ...Factory) (map[component.Type]Factory, error) {
	fMap := map[component.Type]Factory{}
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return fMap, fmt.Errorf("duplicate processor factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}
