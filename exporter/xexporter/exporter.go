// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporter // import "go.opentelemetry.io/collector/exporter/xexporter"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/internal/componentalias"
	"go.opentelemetry.io/collector/pipeline"
)

// Profiles is an exporter that can consume profiles.
type Profiles interface {
	component.Component
	xconsumer.Profiles
}

type Factory interface {
	exporter.Factory

	// CreateProfiles creates a Profiles exporter based on this config.
	// If the exporter type does not support tracing,
	// this function returns the error [pipeline.ErrSignalNotSupported].
	CreateProfiles(ctx context.Context, set exporter.Settings, cfg component.Config) (Profiles, error)

	// ProfilesStability gets the stability level of the Profiles exporter.
	ProfilesStability() component.StabilityLevel
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

// CreateProfilesFunc is the equivalent of Factory.CreateProfiles.
type CreateProfilesFunc func(context.Context, exporter.Settings, component.Config) (Profiles, error)

// WithTraces overrides the default "error not supported" implementation for CreateTraces and the default "undefined" stability level.
func WithTraces(createTraces exporter.CreateTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.opts = append(o.opts, exporter.WithTraces(createTraces, sl))
	})
}

// WithMetrics overrides the default "error not supported" implementation for CreateMetrics and the default "undefined" stability level.
func WithMetrics(createMetrics exporter.CreateMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.opts = append(o.opts, exporter.WithMetrics(createMetrics, sl))
	})
}

// WithLogs overrides the default "error not supported" implementation for CreateLogs and the default "undefined" stability level.
func WithLogs(createLogs exporter.CreateLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.opts = append(o.opts, exporter.WithLogs(createLogs, sl))
	})
}

// WithProfiles overrides the default "error not supported" implementation for CreateProfilesExporter and the default "undefined" stability level.
func WithProfiles(createProfiles CreateProfilesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.profilesStabilityLevel = sl
		o.createProfilesFunc = createProfiles
	})
}

// WithDeprecatedTypeAlias configures a deprecated type alias for the exporter. Only one alias is supported per exporter.
// When the alias is used in configuration, a deprecation warning is automatically logged.
func WithDeprecatedTypeAlias(alias component.Type) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.SetDeprecatedAlias(alias)
	})
}

type factory struct {
	exporter.Factory
	componentalias.TypeAliasHolder
	opts                   []exporter.FactoryOption
	createProfilesFunc     CreateProfilesFunc
	profilesStabilityLevel component.StabilityLevel
}

func (f *factory) ProfilesStability() component.StabilityLevel {
	return f.profilesStabilityLevel
}

func (f *factory) CreateProfiles(ctx context.Context, set exporter.Settings, cfg component.Config) (Profiles, error) {
	if f.createProfilesFunc == nil {
		return nil, pipeline.ErrSignalNotSupported
	}
	if err := componentalias.ValidateComponentType(f.Factory, set.ID); err != nil {
		return nil, err
	}
	return f.createProfilesFunc(ctx, set, cfg)
}

// NewFactory creates a wrapped exporter.Factory with experimental capabilities.
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	f := &factory{TypeAliasHolder: componentalias.NewTypeAliasHolder()}
	for _, opt := range options {
		opt.applyOption(f)
	}
	f.Factory = exporter.NewFactory(cfgType, createDefaultConfig, f.opts...)
	f.Factory.(componentalias.TypeAliasHolder).SetDeprecatedAlias(f.DeprecatedAlias())
	return f
}
