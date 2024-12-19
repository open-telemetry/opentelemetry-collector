// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xprocessor // import "go.opentelemetry.io/collector/processor/xprocessor"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
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

	// CreateEntities creates an Entities processor based on this config.
	// If the processor type does not support entities,
	// this function returns the error [pipeline.ErrSignalNotSupported].
	// Implementers can assume `next` is never nil.
	CreateEntities(ctx context.Context, set processor.Settings, cfg component.Config, next xconsumer.Entities) (Entities, error)

	// EntitiesStability gets the stability level of the Entities processor.
	EntitiesStability() component.StabilityLevel
}

// Profiles is a processor that can consume profiles.
type Profiles interface {
	component.Component
	xconsumer.Profiles
}

// Entities is a processor that can consume entities.
type Entities interface {
	component.Component
	xconsumer.Entities
}

// CreateProfilesFunc is the equivalent of Factory.CreateProfiles().
// CreateProfilesFunc is the equivalent of Factory.CreateProfiles().
type CreateProfilesFunc func(context.Context, processor.Settings, component.Config, xconsumer.Profiles) (Profiles, error)

// CreateProfiles implements Factory.CreateProfiles.
func (f CreateProfilesFunc) CreateProfiles(ctx context.Context, set processor.Settings, cfg component.Config, next xconsumer.Profiles) (Profiles, error) {
	if f == nil {
		return nil, pipeline.ErrSignalNotSupported
	}
	return f(ctx, set, cfg, next)
}

// CreateEntitiesFunc is the equivalent of Factory.CreateEntities().
type CreateEntitiesFunc func(context.Context, processor.Settings, component.Config, xconsumer.Entities) (Entities, error)

// CreateEntities implements Factory.CreateEntities.
func (f CreateEntitiesFunc) CreateEntities(ctx context.Context, set processor.Settings, cfg component.Config, next xconsumer.Entities) (Entities, error) {
	if f == nil {
		return nil, pipeline.ErrSignalNotSupported
	}
	return f(ctx, set, cfg, next)
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
	processor.Factory
	CreateProfilesFunc
	profilesStabilityLevel component.StabilityLevel
	CreateEntitiesFunc
	entitiesStabilityLevel component.StabilityLevel
}

func (f factory) ProfilesStability() component.StabilityLevel {
	return f.profilesStabilityLevel
}

func (f factory) EntitiesStability() component.StabilityLevel {
	return f.entitiesStabilityLevel
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
		o.CreateProfilesFunc = createProfiles
	})
}

// WithEntities overrides the default "error not supported" implementation for CreateEntities and the default "undefined" stability level.
func WithEntities(createEntities CreateEntitiesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.entitiesStabilityLevel = sl
		o.CreateEntitiesFunc = createEntities
	})
}

// NewFactory returns a Factory.
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	opts := factoryOpts{factory: &factory{}}
	for _, opt := range options {
		opt.applyOption(&opts)
	}
	opts.factory.Factory = processor.NewFactory(cfgType, createDefaultConfig, opts.opts...)
	return opts.factory
}
