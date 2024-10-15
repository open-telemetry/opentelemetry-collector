// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiver // import "go.opentelemetry.io/collector/receiver"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pipeline"
)

// Traces receiver receives traces.
// Its purpose is to translate data from any format to the collector's internal trace format.
// Traces receiver feeds a consumer.Traces with data.
//
// For example, it could be Zipkin data source which translates Zipkin spans into ptrace.Traces.
type Traces interface {
	component.Component
}

// Metrics receiver receives metrics.
// Its purpose is to translate data from any format to the collector's internal metrics format.
// Metrics receiver feeds a consumer.Metrics with data.
//
// For example, it could be Prometheus data source which translates Prometheus metrics into pmetric.Metrics.
type Metrics interface {
	component.Component
}

// Logs receiver receives logs.
// Its purpose is to translate data from any format to the collector's internal logs data format.
// Logs receiver feeds a consumer.Logs with data.
//
// For example, it could be a receiver that reads syslogs and convert them into plog.Logs.
type Logs interface {
	component.Component
}

// Settings configures receiver creators.
type Settings struct {
	// ID returns the ID of the component that will be created.
	ID component.ID

	component.TelemetrySettings

	// BuildInfo can be used by components for informational purposes.
	BuildInfo component.BuildInfo
}

// Factory is a factory interface for receivers.
//
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
type Factory interface {
	component.Factory

	// CreateTraces creates a Traces based on this config.
	// If the receiver type does not support traces,
	// this function returns the error [pipeline.ErrSignalNotSupported].
	// Implementers can assume `next` is never nil.
	CreateTraces(ctx context.Context, set Settings, cfg component.Config, next consumer.Traces) (Traces, error)

	// TracesStability gets the stability level of the Traces receiver.
	TracesStability() component.StabilityLevel

	// CreateMetrics creates a Metrics based on this config.
	// If the receiver type does not support metrics,
	// this function returns the error [pipeline.ErrSignalNotSupported].
	// Implementers can assume `next` is never nil.
	CreateMetrics(ctx context.Context, set Settings, cfg component.Config, next consumer.Metrics) (Metrics, error)

	// MetricsStability gets the stability level of the Metrics receiver.
	MetricsStability() component.StabilityLevel

	// CreateLogs creates a Logs based on this config.
	// If the receiver type does not support logs,
	// this function returns the error [pipeline.ErrSignalNotSupported].
	// Implementers can assume `next` is never nil.
	CreateLogs(ctx context.Context, set Settings, cfg component.Config, next consumer.Logs) (Logs, error)

	// LogsStability gets the stability level of the Logs receiver.
	LogsStability() component.StabilityLevel

	unexportedFactoryFunc()
}

// FactoryOption apply changes to Factory.
type FactoryOption interface {
	// applyOption applies the option.
	applyOption(o *factory)
}

// factoryOptionFunc is an FactoryOption created through a function.
type factoryOptionFunc func(*factory)

func (f factoryOptionFunc) applyOption(o *factory) {
	f(o)
}

// CreateTracesFunc is the equivalent of Factory.CreateTraces.
type CreateTracesFunc func(context.Context, Settings, component.Config, consumer.Traces) (Traces, error)

// CreateTraces implements Factory.CreateTraces().
func (f CreateTracesFunc) CreateTraces(ctx context.Context, set Settings, cfg component.Config, next consumer.Traces) (Traces, error) {
	if f == nil {
		return nil, pipeline.ErrSignalNotSupported
	}
	return f(ctx, set, cfg, next)
}

// CreateMetricsFunc is the equivalent of Factory.CreateMetrics.
type CreateMetricsFunc func(context.Context, Settings, component.Config, consumer.Metrics) (Metrics, error)

// CreateMetrics implements Factory.CreateMetrics.
func (f CreateMetricsFunc) CreateMetrics(ctx context.Context, set Settings, cfg component.Config, next consumer.Metrics) (Metrics, error) {
	if f == nil {
		return nil, pipeline.ErrSignalNotSupported
	}
	return f(ctx, set, cfg, next)
}

// CreateLogsFunc is the equivalent of Factory.CreateLogs.
type CreateLogsFunc func(context.Context, Settings, component.Config, consumer.Logs) (Logs, error)

// CreateLogs implements Factory.CreateLogs.
func (f CreateLogsFunc) CreateLogs(ctx context.Context, set Settings, cfg component.Config, next consumer.Logs) (Logs, error) {
	if f == nil {
		return nil, pipeline.ErrSignalNotSupported
	}
	return f(ctx, set, cfg, next)
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

// WithTraces overrides the default "error not supported" implementation for Factory.CreateTraces and the default "undefined" stability level.
func WithTraces(createTraces CreateTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.tracesStabilityLevel = sl
		o.CreateTracesFunc = createTraces
	})
}

// WithMetrics overrides the default "error not supported" implementation for Factory.CreateMetrics and the default "undefined" stability level.
func WithMetrics(createMetrics CreateMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.metricsStabilityLevel = sl
		o.CreateMetricsFunc = createMetrics
	})
}

// WithLogs overrides the default "error not supported" implementation for Factory.CreateLogs and the default "undefined" stability level.
func WithLogs(createLogs CreateLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.logsStabilityLevel = sl
		o.CreateLogsFunc = createLogs
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

// MakeFactoryMap takes a list of receiver factories and returns a map with factory type as keys.
// It returns a non-nil error when there are factories with duplicate type.
func MakeFactoryMap(factories ...Factory) (map[component.Type]Factory, error) {
	fMap := map[component.Type]Factory{}
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return fMap, fmt.Errorf("duplicate receiver factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}
