// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/internal"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
)

// Factory is a factory interface for exporters.
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
type Factory interface {
	component.Factory

	// CreateTracesExporter creates a TracesExporter based on this config.
	// If the exporter type does not support tracing,
	// this function returns the error [pipeline.ErrSignalNotSupported].
	CreateTracesExporter(ctx context.Context, set Settings, cfg component.Config) (Traces, error)

	// TracesExporterStability gets the stability level of the TracesExporter.
	TracesExporterStability() component.StabilityLevel

	// CreateMetricsExporter creates a MetricsExporter based on this config.
	// If the exporter type does not support metrics,
	// this function returns the error [pipeline.ErrSignalNotSupported].
	CreateMetricsExporter(ctx context.Context, set Settings, cfg component.Config) (Metrics, error)

	// MetricsExporterStability gets the stability level of the MetricsExporter.
	MetricsExporterStability() component.StabilityLevel

	// CreateLogsExporter creates a LogsExporter based on the config.
	// If the exporter type does not support logs,
	// this function returns the error [pipeline.ErrSignalNotSupported].
	CreateLogsExporter(ctx context.Context, set Settings, cfg component.Config) (Logs, error)

	// LogsExporterStability gets the stability level of the LogsExporter.
	LogsExporterStability() component.StabilityLevel

	// CreateProfilesExporter creates a ProfilesExporter based on this config.
	// If the exporter type does not support tracing,
	// this function returns the error [pipeline.ErrSignalNotSupported].
	CreateProfilesExporter(ctx context.Context, set Settings, cfg component.Config) (Profiles, error)

	// ProfilesExporterStability gets the stability level of the ProfilesExporter.
	ProfilesExporterStability() component.StabilityLevel

	unexportedFactoryFunc()
}

// FactoryOption apply changes to Factory.
type FactoryOption interface {
	// applyExporterFactoryOption applies the option.
	applyExporterFactoryOption(o *factory)
}

var _ FactoryOption = (*factoryOptionFunc)(nil)

// factoryOptionFunc is an ExporterFactoryOption created through a function.
type factoryOptionFunc func(*factory)

func (f factoryOptionFunc) applyExporterFactoryOption(o *factory) {
	f(o)
}

// CreateTracesFunc is the equivalent of Factory.CreateTraces.
type CreateTracesFunc func(context.Context, Settings, component.Config) (Traces, error)

// CreateTracesExporter implements ExporterFactory.CreateTracesExporter().
func (f CreateTracesFunc) CreateTracesExporter(ctx context.Context, set Settings, cfg component.Config) (Traces, error) {
	if f == nil {
		return nil, pipeline.ErrSignalNotSupported
	}
	return f(ctx, set, cfg)
}

// CreateMetricsFunc is the equivalent of Factory.CreateMetrics.
type CreateMetricsFunc func(context.Context, Settings, component.Config) (Metrics, error)

// CreateMetricsExporter implements ExporterFactory.CreateMetricsExporter().
func (f CreateMetricsFunc) CreateMetricsExporter(ctx context.Context, set Settings, cfg component.Config) (Metrics, error) {
	if f == nil {
		return nil, pipeline.ErrSignalNotSupported
	}
	return f(ctx, set, cfg)
}

// CreateLogsFunc is the equivalent of Factory.CreateLogs.
type CreateLogsFunc func(context.Context, Settings, component.Config) (Logs, error)

// CreateLogsExporter implements Factory.CreateLogsExporter().
func (f CreateLogsFunc) CreateLogsExporter(ctx context.Context, set Settings, cfg component.Config) (Logs, error) {
	if f == nil {
		return nil, pipeline.ErrSignalNotSupported
	}
	return f(ctx, set, cfg)
}

// CreateProfilesFunc is the equivalent of Factory.CreateProfiles.
type CreateProfilesFunc func(context.Context, Settings, component.Config) (Profiles, error)

// CreateProfilesExporter implements ExporterFactory.CreateProfilesExporter().
func (f CreateProfilesFunc) CreateProfilesExporter(ctx context.Context, set Settings, cfg component.Config) (Profiles, error) {
	if f == nil {
		return nil, pipeline.ErrSignalNotSupported
	}
	return f(ctx, set, cfg)
}

type factory struct {
	cfgType component.Type
	component.CreateDefaultConfigFunc
	CreateTracesFunc
	tracesStabilityLevel component.StabilityLevel
	CreateMetricsFunc
	metricsStabilityLevel component.StabilityLevel
	CreateLogsFunc
	logsStabilityLevel component.StabilityLevel
	CreateProfilesFunc
	profilesStabilityLevel component.StabilityLevel
}

func (f *factory) Type() component.Type {
	return f.cfgType
}

func (f *factory) unexportedFactoryFunc() {}

func (f *factory) TracesExporterStability() component.StabilityLevel {
	return f.tracesStabilityLevel
}

func (f *factory) MetricsExporterStability() component.StabilityLevel {
	return f.metricsStabilityLevel
}

func (f *factory) LogsExporterStability() component.StabilityLevel {
	return f.logsStabilityLevel
}

func (f *factory) ProfilesExporterStability() component.StabilityLevel {
	return f.profilesStabilityLevel
}

// WithTraces overrides the default "error not supported" implementation for CreateTracesExporter and the default "undefined" stability level.
func WithTraces(createTraces CreateTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.tracesStabilityLevel = sl
		o.CreateTracesFunc = createTraces
	})
}

// WithMetrics overrides the default "error not supported" implementation for CreateMetricsExporter and the default "undefined" stability level.
func WithMetrics(createMetrics CreateMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.metricsStabilityLevel = sl
		o.CreateMetricsFunc = createMetrics
	})
}

// WithLogs overrides the default "error not supported" implementation for CreateLogsExporter and the default "undefined" stability level.
func WithLogs(createLogs CreateLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.logsStabilityLevel = sl
		o.CreateLogsFunc = createLogs
	})
}

// WithProfiles overrides the default "error not supported" implementation for CreateProfilesExporter and the default "undefined" stability level.
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
		opt.applyExporterFactoryOption(f)
	}
	return f
}
