// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xscraper // import "go.opentelemetry.io/collector/scraper/xscraper"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/scraper"
)

type Factory interface {
	scraper.Factory

	// CreateProfiles creates a Profiles scraper based on this config.
	// If the scraper type does not support profiles,
	// this function returns the error [pipeline.ErrSignalNotSupported].
	CreateProfiles(ctx context.Context, set scraper.Settings, cfg component.Config) (Profiles, error)

	// ProfilesStability gets the stability level of the Profiles scraper.
	ProfilesStability() component.StabilityLevel
}

// FactoryOption apply changes to Options.
type FactoryOption interface {
	// applyOption applies the option.
	applyOption(o *factoryOpts)
}

type factoryOpts struct {
	opts []scraper.FactoryOption
	*factory
}

var _ FactoryOption = (*factoryOptionFunc)(nil)

// factoryOptionFunc is a FactoryOption created through a function.
type factoryOptionFunc func(*factoryOpts)

func (f factoryOptionFunc) applyOption(o *factoryOpts) {
	f(o)
}

type factory struct {
	scraper.Factory

	createProfilesFunc     CreateProfilesFunc
	profilesStabilityLevel component.StabilityLevel
}

func (f *factory) ProfilesStability() component.StabilityLevel {
	return f.profilesStabilityLevel
}

func (f *factory) CreateProfiles(ctx context.Context, set scraper.Settings, cfg component.Config) (Profiles, error) {
	if f.createProfilesFunc == nil {
		return nil, pipeline.ErrSignalNotSupported
	}
	return f.createProfilesFunc(ctx, set, cfg)
}

// WithLogs overrides the default "error not supported" implementation for CreateLogs and the default "undefined" stability level.
func WithLogs(createLogs scraper.CreateLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, scraper.WithLogs(createLogs, sl))
	})
}

// WithMetrics overrides the default "error not supported" implementation for CreateMetrics and the default "undefined" stability level.
func WithMetrics(createMetrics scraper.CreateMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryOpts) {
		o.opts = append(o.opts, scraper.WithMetrics(createMetrics, sl))
	})
}

// CreateProfilesFunc is the equivalent of Factory.CreateProfiles().
type CreateProfilesFunc func(context.Context, scraper.Settings, component.Config) (Profiles, error)

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
	opts.Factory = scraper.NewFactory(cfgType, createDefaultConfig, opts.opts...)
	return opts.factory
}
