// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraper // import "go.opentelemetry.io/collector/scraper"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
)

// Settings configures scraper creators.
type Settings struct {
	// ID returns the ID of the component that will be created.
	ID component.ID

	component.TelemetrySettings

	// BuildInfo can be used by components for informational purposes.
	BuildInfo component.BuildInfo

	// prevent unkeyed literal initialization
	_ struct{}
}

// Factory is factory interface for scrapers.
//
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
type Factory interface {
	component.Factory

	// CreateLogs creates a Logs scraper based on this config.
	// If the scraper type does not support logs,
	// this function returns the error [pipeline.ErrSignalNotSupported].
	CreateLogs(ctx context.Context, set Settings, cfg component.Config) (Logs, error)

	// CreateMetrics creates a Metrics scraper based on this config.
	// If the scraper type does not support metrics,
	// this function returns the error [pipeline.ErrSignalNotSupported].
	CreateMetrics(ctx context.Context, set Settings, cfg component.Config) (Metrics, error)

	// LogsStability gets the stability level of the Logs scraper.
	LogsStability() component.StabilityLevel

	// MetricsStability gets the stability level of the Metrics scraper.
	MetricsStability() component.StabilityLevel

	unexportedFactoryFunc()
}

// FactoryOption apply changes to Options.
type FactoryOption interface {
	// applyOption applies the option.
	applyOption(o *factory)
}

var _ FactoryOption = (*factoryOptionFunc)(nil)

// factoryOptionFunc is a FactoryOption created through a function.
type factoryOptionFunc func(*factory)

func (f factoryOptionFunc) applyOption(o *factory) {
	f(o)
}

type factory struct {
	cfgType component.Type
	component.CreateDefaultConfigFunc
	createLogsFunc        CreateLogsFunc
	createMetricsFunc     CreateMetricsFunc
	logsStabilityLevel    component.StabilityLevel
	metricsStabilityLevel component.StabilityLevel
}

func (f *factory) Type() component.Type {
	return f.cfgType
}

func (f *factory) unexportedFactoryFunc() {}

func (f *factory) LogsStability() component.StabilityLevel {
	return f.logsStabilityLevel
}

func (f *factory) MetricsStability() component.StabilityLevel {
	return f.metricsStabilityLevel
}

func (f *factory) CreateLogs(ctx context.Context, set Settings, cfg component.Config) (Logs, error) {
	if f.createLogsFunc == nil {
		return nil, pipeline.ErrSignalNotSupported
	}
	return f.createLogsFunc(ctx, set, cfg)
}

func (f *factory) CreateMetrics(ctx context.Context, set Settings, cfg component.Config) (Metrics, error) {
	if f.createMetricsFunc == nil {
		return nil, pipeline.ErrSignalNotSupported
	}
	return f.createMetricsFunc(ctx, set, cfg)
}

// CreateLogsFunc is the equivalent of Factory.CreateLogs().
type CreateLogsFunc func(context.Context, Settings, component.Config) (Logs, error)

// CreateMetricsFunc is the equivalent of Factory.CreateMetrics().
type CreateMetricsFunc func(context.Context, Settings, component.Config) (Metrics, error)

// WithLogs overrides the default "error not supported" implementation for CreateLogs and the default "undefined" stability level.
func WithLogs(createLogs CreateLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.logsStabilityLevel = sl
		o.createLogsFunc = createLogs
	})
}

// WithMetrics overrides the default "error not supported" implementation for CreateMetrics and the default "undefined" stability level.
func WithMetrics(createMetrics CreateMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.metricsStabilityLevel = sl
		o.createMetricsFunc = createMetrics
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
