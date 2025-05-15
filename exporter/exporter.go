// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporter // import "go.opentelemetry.io/collector/exporter"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter/internal/experr"
	"go.opentelemetry.io/collector/pipeline"
)

// Traces is an exporter that can consume traces.
type Traces interface {
	component.Component
	consumer.Traces
}

// Metrics is an exporter that can consume metrics.
type Metrics interface {
	component.Component
	consumer.Metrics
}

// Logs is an exporter that can consume logs.
type Logs interface {
	component.Component
	consumer.Logs
}

// Settings configures exporter creators.
type Settings struct {
	// ID returns the ID of the component that will be created.
	ID component.ID

	component.TelemetrySettings

	// BuildInfo can be used by components for informational purposes
	BuildInfo component.BuildInfo

	// prevent unkeyed literal initialization
	_ struct{}
}

// Factory is factory interface for exporters.
//
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
type Factory interface {
	component.Factory

	// CreateTraces creates a Traces exporter based on this config.
	// If the exporter type does not support tracing,
	// this function returns the error [pipeline.ErrSignalNotSupported].
	CreateTraces(ctx context.Context, set Settings, cfg component.Config) (Traces, error)

	// TracesStability gets the stability level of the Traces exporter.
	TracesStability() component.StabilityLevel

	// CreateMetrics creates a Metrics exporter based on this config.
	// If the exporter type does not support metrics,
	// this function returns the error [pipeline.ErrSignalNotSupported].
	CreateMetrics(ctx context.Context, set Settings, cfg component.Config) (Metrics, error)

	// MetricsStability gets the stability level of the Metrics exporter.
	MetricsStability() component.StabilityLevel

	// CreateLogs creates a Logs exporter based on the config.
	// If the exporter type does not support logs,
	// this function returns the error [pipeline.ErrSignalNotSupported].
	CreateLogs(ctx context.Context, set Settings, cfg component.Config) (Logs, error)

	// LogsStability gets the stability level of the Logs exporter.
	LogsStability() component.StabilityLevel

	unexportedFactoryFunc()
}

// FactoryOption apply changes to Factory.
type FactoryOption interface {
	// applyOption applies the option.
	applyOption(o *factory)
}

var _ FactoryOption = (*factoryOptionFunc)(nil)

// factoryOptionFunc is an FactoryOption created through a function.
type factoryOptionFunc func(*factory)

func (f factoryOptionFunc) applyOption(o *factory) {
	f(o)
}

// CreateTracesFunc is the equivalent of Factory.CreateTraces.
type CreateTracesFunc func(context.Context, Settings, component.Config) (Traces, error)

// CreateMetricsFunc is the equivalent of Factory.CreateMetrics.
type CreateMetricsFunc func(context.Context, Settings, component.Config) (Metrics, error)

// CreateLogsFunc is the equivalent of Factory.CreateLogs.
type CreateLogsFunc func(context.Context, Settings, component.Config) (Logs, error)

type factory struct {
	cfgType component.Type
	component.CreateDefaultConfigFunc
	createTracesFunc      CreateTracesFunc
	tracesStabilityLevel  component.StabilityLevel
	createMetricsFunc     CreateMetricsFunc
	metricsStabilityLevel component.StabilityLevel
	createLogsFunc        CreateLogsFunc
	logsStabilityLevel    component.StabilityLevel
}

func (f *factory) Type() component.Type {
	return f.cfgType
}

func (f *factory) unexportedFactoryFunc() {}

func (f *factory) TracesStability() component.StabilityLevel {
	return f.tracesStabilityLevel
}

func (f *factory) MetricsStability() component.StabilityLevel {
	return f.metricsStabilityLevel
}

func (f *factory) LogsStability() component.StabilityLevel {
	return f.logsStabilityLevel
}

func (f *factory) CreateTraces(ctx context.Context, set Settings, cfg component.Config) (Traces, error) {
	if f.createTracesFunc == nil {
		return nil, pipeline.ErrSignalNotSupported
	}

	if set.ID.Type() != f.Type() {
		return nil, experr.ErrIDMismatch(set.ID, f.Type())
	}

	return f.createTracesFunc(ctx, set, cfg)
}

func (f *factory) CreateMetrics(ctx context.Context, set Settings, cfg component.Config) (Metrics, error) {
	if f.createMetricsFunc == nil {
		return nil, pipeline.ErrSignalNotSupported
	}

	if set.ID.Type() != f.Type() {
		return nil, experr.ErrIDMismatch(set.ID, f.Type())
	}

	return f.createMetricsFunc(ctx, set, cfg)
}

func (f *factory) CreateLogs(ctx context.Context, set Settings, cfg component.Config) (Logs, error) {
	if f.createLogsFunc == nil {
		return nil, pipeline.ErrSignalNotSupported
	}

	if set.ID.Type() != f.Type() {
		return nil, experr.ErrIDMismatch(set.ID, f.Type())
	}

	return f.createLogsFunc(ctx, set, cfg)
}

// WithTraces overrides the default "error not supported" implementation for Factory.CreateTraces and the default "undefined" stability level.
func WithTraces(createTraces CreateTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.tracesStabilityLevel = sl
		o.createTracesFunc = createTraces
	})
}

// WithMetrics overrides the default "error not supported" implementation for Factory.CreateMetrics and the default "undefined" stability level.
func WithMetrics(createMetrics CreateMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.metricsStabilityLevel = sl
		o.createMetricsFunc = createMetrics
	})
}

// WithLogs overrides the default "error not supported" implementation for Factory.CreateLogs and the default "undefined" stability level.
func WithLogs(createLogs CreateLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.logsStabilityLevel = sl
		o.createLogsFunc = createLogs
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
