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
// ProfilessReceiver feeds a consumerprofiles.Profiles with data.
//
// For example, it could be a pprof data source which translates pprof profiles into pprofile.Profiles.
type Profiles interface {
	component.Component
}

// Factory is a factory interface for receivers.
//
// This interface cannot be directly implemented. Implementations must
// use the NewReceiverFactory to implement it.
type Factory interface {
	receiver.Factory

	// CreateProfilesReceiver creates a ProfilesReceiver based on this config.
	// If the receiver type does not support tracing or if the config is not valid
	// an error will be returned instead. `nextConsumer` is never nil.
	CreateProfilesReceiver(ctx context.Context, set receiver.Settings, cfg component.Config, nextConsumer consumerprofiles.Profiles) (Profiles, error)

	// ProfilesReceiverStability gets the stability level of the ProfilesReceiver.
	ProfilesReceiverStability() component.StabilityLevel
}

// CreateProfilesFunc is the equivalent of Factory.CreateProfiles.
type CreateProfilesFunc func(context.Context, receiver.Settings, component.Config, consumerprofiles.Profiles) (Profiles, error)

// CreateProfilesReceiver implements Factory.CreateProfilesReceiver().
func (f CreateProfilesFunc) CreateProfilesReceiver(
	ctx context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumerprofiles.Profiles) (Profiles, error) {
	if f == nil {
		return nil, pipeline.ErrSignalNotSupported
	}
	return f(ctx, set, cfg, nextConsumer)
}

// FactoryOption apply changes to ReceiverOptions.
type FactoryOption interface {
	// applyOption applies the option.
	applyOption(o *factoryOpts)
}

// factoryOptionFunc is an ReceiverFactoryOption created through a function.
type factoryOptionFunc func(*factoryOpts)

func (f factoryOptionFunc) applyOption(o *factoryOpts) {
	f(o)
}

type factory struct {
	receiver.Factory
	CreateProfilesFunc
	profilesStabilityLevel component.StabilityLevel
}

func (f *factory) ProfilesReceiverStability() component.StabilityLevel {
	return f.profilesStabilityLevel
}

type factoryOpts struct {
	cfgType component.Type
	component.CreateDefaultConfigFunc
	opts []receiver.FactoryOption
	CreateProfilesFunc
	profilesStabilityLevel component.StabilityLevel
}

// WithTraces overrides the default "error not supported" implementation for CreateTracesReceiver and the default "undefined" stability level.
func WithTraces(createTracesReceiver receiver.CreateTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, receiver.WithTraces(createTracesReceiver, sl))
	})
}

// WithMetrics overrides the default "error not supported" implementation for CreateMetricsReceiver and the default "undefined" stability level.
func WithMetrics(createMetricsReceiver receiver.CreateMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, receiver.WithMetrics(createMetricsReceiver, sl))
	})
}

// WithLogs overrides the default "error not supported" implementation for CreateLogsReceiver and the default "undefined" stability level.
func WithLogs(createLogsReceiver receiver.CreateLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, receiver.WithLogs(createLogsReceiver, sl))
	})
}

// WithProfiles overrides the default "error not supported" implementation for CreateProfilesReceiver and the default "undefined" stability level.
func WithProfiles(createProfilesReceiver CreateProfilesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.profilesStabilityLevel = sl
		o.CreateProfilesFunc = createProfilesReceiver
	})
}

// NewFactory returns a Factory.
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	opts := factoryOpts{
		cfgType:                 cfgType,
		CreateDefaultConfigFunc: createDefaultConfig,
	}
	for _, opt := range options {
		opt.applyOption(&opts)
	}
	return &factory{
		Factory:                receiver.NewFactory(opts.cfgType, opts.CreateDefaultConfig, opts.opts...),
		CreateProfilesFunc:     opts.CreateProfilesFunc,
		profilesStabilityLevel: opts.profilesStabilityLevel,
	}
}
