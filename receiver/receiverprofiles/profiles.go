// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiverprofiles // import "go.opentelemetry.io/collector/receiver/receiverprofiles"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/receiver"
)

// Profiles receiver receives profiles.
// Its purpose is to translate data from any format to the collector's internal profile format.
// Profiles receiver feeds a consumerprofiles.Profiles with data.
//
// For example, it could be a pprof data source which translates pprof profiles into pprofile.Profiles.
type Profiles interface {
	component.Component
}

// Factory is a factory interface for receivers.
//
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
type Factory interface {
	receiver.Factory

	// CreateProfiles creates a Profiles based on this config.
	// If the receiver type does not support tracing or if the config is not valid
	// an error will be returned instead. `next` is never nil.
	CreateProfiles(ctx context.Context, set receiver.Settings, cfg component.Config, next consumerprofiles.Profiles) (Profiles, error)

	// ProfilesStability gets the stability level of the Profiles receiver.
	ProfilesStability() component.StabilityLevel
}

// CreateProfilesFunc is the equivalent of Factory.CreateProfiles.
type CreateProfilesFunc func(context.Context, receiver.Settings, component.Config, consumerprofiles.Profiles) (Profiles, error)

// CreateProfiles implements Factory.CreateProfiles.
func (f CreateProfilesFunc) CreateProfiles(ctx context.Context, set receiver.Settings, cfg component.Config, next consumerprofiles.Profiles) (Profiles, error) {
	if f == nil {
		return nil, pipeline.ErrSignalNotSupported
	}
	return f(ctx, set, cfg, next)
}

// FactoryOption apply changes to Factory.
type FactoryOption interface {
	// applyOption applies the option.
	applyOption(o *factoryOpts)
}

// factoryOptionFunc is a FactoryOption created through a function.
type factoryOptionFunc func(*factoryOpts)

func (f factoryOptionFunc) applyOption(o *factoryOpts) {
	f(o)
}

type factory struct {
	receiver.Factory
	CreateProfilesFunc
	profilesStabilityLevel component.StabilityLevel
}

func (f *factory) ProfilesStability() component.StabilityLevel {
	return f.profilesStabilityLevel
}

type factoryOpts struct {
	opts []receiver.FactoryOption
	*factory
}

// WithTraces overrides the default "error not supported" implementation for Factory.CreateTraces and the default "undefined" stability level.
func WithTraces(createTraces receiver.CreateTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, receiver.WithTraces(createTraces, sl))
	})
}

// WithMetrics overrides the default "error not supported" implementation for Factory.CreateMetrics and the default "undefined" stability level.
func WithMetrics(createMetrics receiver.CreateMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, receiver.WithMetrics(createMetrics, sl))
	})
}

// WithLogs overrides the default "error not supported" implementation for Factory.CreateLogs and the default "undefined" stability level.
func WithLogs(createLogs receiver.CreateLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, receiver.WithLogs(createLogs, sl))
	})
}

// WithProfiles overrides the default "error not supported" implementation for Factory.CreateProfiles and the default "undefined" stability level.
func WithProfiles(createProfiles CreateProfilesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.profilesStabilityLevel = sl
		o.CreateProfilesFunc = createProfiles
	})
}

// NewFactory returns a Factory.
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	opts := factoryOpts{factory: &factory{}}
	for _, opt := range options {
		opt.applyOption(&opts)
	}
	opts.factory.Factory = receiver.NewFactory(cfgType, createDefaultConfig, opts.opts...)
	return opts.factory
}
