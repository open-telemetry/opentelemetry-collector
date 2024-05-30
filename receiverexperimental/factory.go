// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiverexperimental // import "go.opentelemetry.io/collector/receiverexperimental"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumerexperimental"
	"go.opentelemetry.io/collector/receiver"
) // Factory is factory interface for receivers.
// This interface cannot be directly implemented. Implementations must
// use the NewReceiverFactory to implement it.
type Factory interface {
	component.Factory

	// CreateProfilesReceiver creates a ProfilesReceiver based on this config.
	// If the receiver type does not support tracing or if the config is not valid
	// an error will be returned instead. `nextConsumer` is never nil.
	CreateProfilesReceiver(ctx context.Context, set receiver.CreateSettings, cfg component.Config, nextConsumer consumerexperimental.Profiles) (Profiles, error)

	// ProfilesReceiverStability gets the stability level of the ProfilesReceiver.
	ProfilesReceiverStability() component.StabilityLevel

	unexportedFactoryFunc()
}

// FactoryOption apply changes to ReceiverOptions.
type FactoryOption interface {
	// applyOption applies the option.
	applyOption(o *factory)
}

// factoryOptionFunc is an ReceiverFactoryOption created through a function.
type factoryOptionFunc func(*factory)

func (f factoryOptionFunc) applyOption(o *factory) {
	f(o)
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

func (f *factory) ProfilesReceiverStability() component.StabilityLevel {
	return f.profilesStabilityLevel
}

// WithProfiles overrides the default "error not supported" implementation for CreateProfilesReceiver and the default "undefined" stability level.
func WithProfiles(createProfilesReceiver CreateProfilesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.profilesStabilityLevel = sl
		o.CreateProfilesFunc = createProfilesReceiver
	})
}

// NewFactory returns a Factory.
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	f := &factory{
		cfgType:                 cfgType,
		CreateDefaultConfigFunc: createDefaultConfig,
	}
	for _, opt := range options {
		opt.applyOption(f)
	}
	return f
}

// MakeFactoryMap takes a list of receiver factories and returns a map with factory type as keys.
// It returns a non-nil error when there are factories with duplicate type.
func MakeFactoryMap(factories ...Factory) (map[component.Type]Factory, error) {
	fMap := map[component.Type]Factory{}
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return fMap, fmt.Errorf("duplicate receiver factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}
