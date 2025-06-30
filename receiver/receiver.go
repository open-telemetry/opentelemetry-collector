// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiver // import "go.opentelemetry.io/collector/receiver"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/receiver/internal"
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

	// prevent unkeyed literal initialization
	_ struct{}
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
	applyOption(o *factoryImpl, cfgType component.Type)
}

// factoryOptionFunc is an FactoryOption created through a function.
type factoryOptionFunc func(*factoryImpl, component.Type)

func (f factoryOptionFunc) applyOption(o *factoryImpl, cfgType component.Type) {
	f(o, cfgType)
}

// CreateTracesFunc is the equivalent of Factory.CreateTraces.
type CreateTracesFunc func(context.Context, Settings, component.Config, consumer.Traces) (Traces, error)

// CreateMetricsFunc is the equivalent of Factory.CreateMetrics.
type CreateMetricsFunc func(context.Context, Settings, component.Config, consumer.Metrics) (Metrics, error)

// CreateLogsFunc is the equivalent of Factory.CreateLogs.
type CreateLogsFunc func(context.Context, Settings, component.Config, consumer.Logs) (Logs, error)

// TracesStabilityFunc is a functional way to construct Factory implementations.
type TracesStabilityFunc func() component.StabilityLevel

// MetricsStabilityFunc is a functional way to construct Factory implementations.
type MetricsStabilityFunc func() component.StabilityLevel

// LogsStabilityFunc is a functional way to construct Factory implementations.
type LogsStabilityFunc func() component.StabilityLevel

type factoryImpl struct {
	component.Factory
	CreateTracesFunc
	TracesStabilityFunc
	CreateMetricsFunc
	MetricsStabilityFunc
	CreateLogsFunc
	LogsStabilityFunc
}

var _ Factory = factoryImpl{}

func (f factoryImpl) unexportedFactoryFunc() {}

func (f TracesStabilityFunc) TracesStability() component.StabilityLevel {
	if f == nil {
		return component.StabilityLevelUndefined
	}
	return f()
}

func (f MetricsStabilityFunc) MetricsStability() component.StabilityLevel {
	if f == nil {
		return component.StabilityLevelUndefined
	}
	return f()
}

func (f LogsStabilityFunc) LogsStability() component.StabilityLevel {
	if f == nil {
		return component.StabilityLevelUndefined
	}
	return f()
}

type creator[A, B any] func(ctx context.Context, set Settings, cfg component.Config, next A) (B, error)

func typeChecked[A, B any](cf creator[A, B], cfgType component.Type) creator[A, B] {
	return func(ctx context.Context, set Settings, cfg component.Config, next A) (B, error) {
		if set.ID.Type() != cfgType {
			var zero B
			return zero, internal.ErrIDMismatch(set.ID, cfgType)
		}
		return cf(ctx, set, cfg, next)
	}
}

func (f CreateTracesFunc) CreateTraces(ctx context.Context, set Settings, cfg component.Config, next consumer.Traces) (Traces, error) {
	if f == nil {
		return nil, pipeline.ErrSignalNotSupported
	}

	return f(ctx, set, cfg, next)
}

func (f CreateMetricsFunc) CreateMetrics(ctx context.Context, set Settings, cfg component.Config, next consumer.Metrics) (Metrics, error) {
	if f == nil {
		return nil, pipeline.ErrSignalNotSupported
	}

	return f(ctx, set, cfg, next)
}

func (f CreateLogsFunc) CreateLogs(ctx context.Context, set Settings, cfg component.Config, next consumer.Logs) (Logs, error) {
	if f == nil {
		return nil, pipeline.ErrSignalNotSupported
	}

	return f(ctx, set, cfg, next)
}

// WithTraces overrides the default "error not supported" implementation for Factory.CreateTraces and the default "undefined" stability level.
func WithTraces(createTraces CreateTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryImpl, cfgType component.Type) {
		o.TracesStabilityFunc = sl.Self
		o.CreateTracesFunc = CreateTracesFunc(typeChecked[consumer.Traces, Traces](creator[consumer.Traces, Traces](createTraces), cfgType))
	})
}

// WithMetrics overrides the default "error not supported" implementation for Factory.CreateMetrics and the default "undefined" stability level.
func WithMetrics(createMetrics CreateMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryImpl, cfgType component.Type) {
		o.MetricsStabilityFunc = sl.Self
		o.CreateMetricsFunc = CreateMetricsFunc(typeChecked[consumer.Metrics, Metrics](creator[consumer.Metrics, Metrics](createMetrics), cfgType))
	})
}

// WithLogs overrides the default "error not supported" implementation for Factory.CreateLogs and the default "undefined" stability level.
func WithLogs(createLogs CreateLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryImpl, cfgType component.Type) {
		o.LogsStabilityFunc = sl.Self
		o.CreateLogsFunc = CreateLogsFunc(typeChecked[consumer.Logs, Logs](creator[consumer.Logs, Logs](createLogs), cfgType))
	})
}

// NewFactory returns a Factory.
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	f := factoryImpl{
		Factory: component.NewFactoryImpl(cfgType.Self, createDefaultConfig),
	}
	for _, opt := range options {
		opt.applyOption(&f, cfgType)
	}
	return f
}

func NewFactoryImpl(
	factory component.Factory,
	tracesFunc CreateTracesFunc,
	tracesStab TracesStabilityFunc,
	metricsFunc CreateMetricsFunc,
	mtricsStab MetricsStabilityFunc,
	createLogs CreateLogsFunc,
	logsStab LogsStabilityFunc,
) Factory {
	return factoryImpl{
		Factory:              factory,
		CreateTracesFunc:     tracesFunc,
		TracesStabilityFunc:  tracesStab,
		CreateMetricsFunc:    metricsFunc,
		MetricsStabilityFunc: mtricsStab,
		CreateLogsFunc:       createLogs,
		LogsStabilityFunc:    logsStab,
	}
}
