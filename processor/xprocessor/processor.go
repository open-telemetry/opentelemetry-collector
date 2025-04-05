// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xprocessor // import "go.opentelemetry.io/collector/processor/xprocessor"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/internal"
)

// Factory is a component.Factory interface for processors.
//
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
type Factory interface {
	processor.Factory

	// CreateProfiles creates a Profiles processor based on this config.
	// If the processor type does not support tracing or if the config is not valid,
	// an error will be returned instead.
	CreateProfiles(ctx context.Context, set processor.Settings, cfg component.Config, next xconsumer.Profiles) (Profiles, error)

	// ProfilesStability gets the stability level of the Profiles processor.
	ProfilesStability() component.StabilityLevel
}

// Profiles is a processor that can consume profiles.
type Profiles interface {
	component.Component
	xconsumer.Profiles
}

// CreateProfilesFunc is the equivalent of Factory.CreateProfiles().
// CreateProfilesFunc is the equivalent of Factory.CreateProfiles().
type CreateProfilesFunc func(context.Context, processor.Settings, component.Config, xconsumer.Profiles) (Profiles, error)

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
	processor.Factory
	createProfilesFunc     CreateProfilesFunc
	profilesStabilityLevel component.StabilityLevel
}

func (f factory) ProfilesStability() component.StabilityLevel {
	return f.profilesStabilityLevel
}

func (f factory) CreateProfiles(ctx context.Context, set processor.Settings, cfg component.Config, next xconsumer.Profiles) (Profiles, error) {
	if f.createProfilesFunc == nil {
		return nil, pipeline.ErrSignalNotSupported
	}
	if set.ID.Type() != f.Type() {
		return nil, internal.ErrIDMismatch(set.ID, f.Type())
	}
	return f.createProfilesFunc(ctx, set, cfg, next)
}

type factoryOpts struct {
	opts []processor.FactoryOption
	*factory
}

// WithTraces overrides the default "error not supported" implementation for CreateTraces and the default "undefined" stability level.
func WithTraces(createTraces processor.CreateTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, processor.WithTraces(createTraces, sl))
	})
}

// WithMetrics overrides the default "error not supported" implementation for CreateMetrics and the default "undefined" stability level.
func WithMetrics(createMetrics processor.CreateMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, processor.WithMetrics(createMetrics, sl))
	})
}

// WithLogs overrides the default "error not supported" implementation for CreateLogs and the default "undefined" stability level.
func WithLogs(createLogs processor.CreateLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, processor.WithLogs(createLogs, sl))
	})
}

// WithProfiles overrides the default "error not supported" implementation for CreateProfiles and the default "undefined" stability level.
func WithProfiles(createProfiles CreateProfilesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.profilesStabilityLevel = sl
		o.createProfilesFunc = createProfiles
	})
}

// NewFactory returns a Factory.
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	opts := factoryOpts{factory: &factory{}}
	for _, opt := range options {
		opt.applyOption(&opts)
	}
	opts.Factory = processor.NewFactory(cfgType, createDefaultConfig, opts.opts...)
	return opts.factory
}
