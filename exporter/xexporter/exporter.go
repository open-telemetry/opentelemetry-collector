// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporter // import "go.opentelemetry.io/collector/exporter/xexporter"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pipeline"
)

// Profiles is an exporter that can consume profiles.
type Profiles interface {
	component.Component
	xconsumer.Profiles
}

// Entities is an exporter that can consume entities.
type Entities interface {
	component.Component
	xconsumer.Entities
}

type Factory interface {
	exporter.Factory

	// CreateProfiles creates a Profiles exporter based on this config.
	// If the exporter type does not support tracing,
	// this function returns the error [pipeline.ErrSignalNotSupported].
	CreateProfiles(ctx context.Context, set exporter.Settings, cfg component.Config) (Profiles, error)

	// ProfilesStability gets the stability level of the Profiles exporter.
	ProfilesStability() component.StabilityLevel

	// CreateEntities creates an EntitiesExporter based on the config.
	// If the exporter type does not support entities,
	// this function returns the error [pipeline.ErrSignalNotSupported].
	CreateEntities(ctx context.Context, set exporter.Settings, cfg component.Config) (Entities, error)

	// EntitiesStability gets the stability level of the EntityExporter.
	EntitiesStability() component.StabilityLevel
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

type factoryOpts struct {
	opts []exporter.FactoryOption
	*factory
}

// CreateProfilesFunc is the equivalent of Factory.CreateProfiles.
type CreateProfilesFunc func(context.Context, exporter.Settings, component.Config) (Profiles, error)

// CreateProfiles implements Factory.CreateProfiles.
func (f CreateProfilesFunc) CreateProfiles(ctx context.Context, set exporter.Settings, cfg component.Config) (Profiles, error) {
	if f == nil {
		return nil, pipeline.ErrSignalNotSupported
	}
	return f(ctx, set, cfg)
}

// CreateEntitiesFunc is the equivalent of Factory.CreateEntities.
type CreateEntitiesFunc func(context.Context, exporter.Settings, component.Config) (Entities, error)

// CreateEntities implements Factory.CreateEntities.
func (f CreateEntitiesFunc) CreateEntities(ctx context.Context, set exporter.Settings, cfg component.Config) (Entities,
	error,
) {
	if f == nil {
		return nil, pipeline.ErrSignalNotSupported
	}
	return f(ctx, set, cfg)
}

// WithTraces overrides the default "error not supported" implementation for CreateTraces and the default "undefined" stability level.
func WithTraces(createTraces exporter.CreateTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, exporter.WithTraces(createTraces, sl))
	})
}

// WithMetrics overrides the default "error not supported" implementation for CreateMetrics and the default "undefined" stability level.
func WithMetrics(createMetrics exporter.CreateMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, exporter.WithMetrics(createMetrics, sl))
	})
}

// WithLogs overrides the default "error not supported" implementation for CreateLogs and the default "undefined" stability level.
func WithLogs(createLogs exporter.CreateLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, exporter.WithLogs(createLogs, sl))
	})
}

// WithProfiles overrides the default "error not supported" implementation for CreateProfilesExporter and the default "undefined" stability level.
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

type factory struct {
	exporter.Factory

	CreateProfilesFunc
	CreateEntitiesFunc

	profilesStabilityLevel component.StabilityLevel
	entitiesStabilityLevel component.StabilityLevel
}

func (f *factory) ProfilesStability() component.StabilityLevel {
	return f.profilesStabilityLevel
}

func (f *factory) EntitiesStability() component.StabilityLevel {
	return f.entitiesStabilityLevel
}

// NewFactory returns a Factory.
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	opts := factoryOpts{factory: &factory{}}
	for _, opt := range options {
		opt.applyOption(&opts)
	}
	opts.factory.Factory = exporter.NewFactory(cfgType, createDefaultConfig, opts.opts...)
	return opts.factory
}
