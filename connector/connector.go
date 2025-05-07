// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector/internal"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pipeline"
)

// A Traces connector acts as an exporter from a traces pipeline and a receiver
// to one or more traces, metrics, or logs pipelines.
// Traces feeds a consumer.Traces, consumer.Metrics, or consumer.Logs with data.
//
// Examples:
//   - Traces could be collected in one pipeline and routed to another traces pipeline
//     based on criteria such as attributes or other content of the trace. The second
//     pipeline can then process and export the trace to the appropriate backend.
//   - Traces could be summarized by a metrics connector that emits statistics describing
//     the number of traces observed.
//   - Traces could be analyzed by a logs connector that emits events when particular
//     criteria are met.
type Traces interface {
	component.Component
	consumer.Traces
}

// A Metrics connector acts as an exporter from a metrics pipeline and a receiver
// to one or more traces, metrics, or logs pipelines.
// Metrics feeds a consumer.Traces, consumer.Metrics, or consumer.Logs with data.
//
// Examples:
//   - Latency between related data points could be modeled and emitted as traces.
//   - Metrics could be collected in one pipeline and routed to another metrics pipeline
//     based on criteria such as attributes or other content of the metric. The second
//     pipeline can then process and export the metric to the appropriate backend.
//   - Metrics could be analyzed by a logs connector that emits events when particular
//     criteria are met.
type Metrics interface {
	component.Component
	consumer.Metrics
}

// A Logs connector acts as an exporter from a logs pipeline and a receiver
// to one or more traces, metrics, or logs pipelines.
// Logs feeds a consumer.Traces, consumer.Metrics, or consumer.Logs with data.
//
// Examples:
//   - Structured logs containing span information could be consumed and emitted as traces.
//   - Metrics could be extracted from structured logs that contain numeric data.
//   - Logs could be collected in one pipeline and routed to another logs pipeline
//     based on criteria such as attributes or other content of the log. The second
//     pipeline can then process and export the log to the appropriate backend.
type Logs interface {
	component.Component
	consumer.Logs
}

// Settings configures Connector creators.
type Settings struct {
	// ID returns the ID of the component that will be created.
	ID component.ID

	component.TelemetrySettings

	// BuildInfo can be used by components for informational purposes
	BuildInfo component.BuildInfo
	// prevent unkeyed literal initialization
	_ struct{}
}

// Factory is a factory interface for connectors.
//
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
type Factory interface {
	component.Factory

	// CreateDefaultConfig creates the default configuration for the Connector.
	// This method can be called multiple times depending on the pipeline
	// configuration and should not cause side-effects that prevent the creation
	// of multiple instances of the Connector.
	// The object returned by this method needs to pass the checks implemented by
	// 'configtest.CheckConfigStruct'. It is recommended to have these checks in the
	// tests of any implementation of the Factory interface.
	CreateDefaultConfig() component.Config

	CreateTracesToTraces(ctx context.Context, set Settings, cfg component.Config, next consumer.Traces) (Traces, error)
	CreateTracesToMetrics(ctx context.Context, set Settings, cfg component.Config, next consumer.Metrics) (Traces, error)
	CreateTracesToLogs(ctx context.Context, set Settings, cfg component.Config, next consumer.Logs) (Traces, error)

	CreateMetricsToTraces(ctx context.Context, set Settings, cfg component.Config, next consumer.Traces) (Metrics, error)
	CreateMetricsToMetrics(ctx context.Context, set Settings, cfg component.Config, next consumer.Metrics) (Metrics, error)
	CreateMetricsToLogs(ctx context.Context, set Settings, cfg component.Config, next consumer.Logs) (Metrics, error)

	CreateLogsToTraces(ctx context.Context, set Settings, cfg component.Config, next consumer.Traces) (Logs, error)
	CreateLogsToMetrics(ctx context.Context, set Settings, cfg component.Config, next consumer.Metrics) (Logs, error)
	CreateLogsToLogs(ctx context.Context, set Settings, cfg component.Config, next consumer.Logs) (Logs, error)

	TracesToTracesStability() component.StabilityLevel
	TracesToMetricsStability() component.StabilityLevel
	TracesToLogsStability() component.StabilityLevel

	MetricsToTracesStability() component.StabilityLevel
	MetricsToMetricsStability() component.StabilityLevel
	MetricsToLogsStability() component.StabilityLevel

	LogsToTracesStability() component.StabilityLevel
	LogsToMetricsStability() component.StabilityLevel
	LogsToLogsStability() component.StabilityLevel

	unexportedFactoryFunc()
}

// FactoryOption applies changes to Factory.
type FactoryOption interface {
	// apply applies the option.
	apply(o *factory)
}

var _ FactoryOption = (*factoryOptionFunc)(nil)

// factoryOptionFunc is an FactoryOption created through a function.
type factoryOptionFunc func(*factory)

func (f factoryOptionFunc) apply(o *factory) {
	f(o)
}

// CreateTracesToTracesFunc is the equivalent of Factory.CreateTracesToTraces().
type CreateTracesToTracesFunc func(context.Context, Settings, component.Config, consumer.Traces) (Traces, error)

// CreateTracesToMetricsFunc is the equivalent of Factory.CreateTracesToMetrics().
type CreateTracesToMetricsFunc func(context.Context, Settings, component.Config, consumer.Metrics) (Traces, error)

// CreateTracesToLogsFunc is the equivalent of Factory.CreateTracesToLogs().
type CreateTracesToLogsFunc func(context.Context, Settings, component.Config, consumer.Logs) (Traces, error)

// CreateMetricsToTracesFunc is the equivalent of Factory.CreateMetricsToTraces().
type CreateMetricsToTracesFunc func(context.Context, Settings, component.Config, consumer.Traces) (Metrics, error)

// CreateMetricsToMetricsFunc is the equivalent of Factory.CreateMetricsToTraces().
type CreateMetricsToMetricsFunc func(context.Context, Settings, component.Config, consumer.Metrics) (Metrics, error)

// CreateMetricsToLogsFunc is the equivalent of Factory.CreateMetricsToLogs().
type CreateMetricsToLogsFunc func(context.Context, Settings, component.Config, consumer.Logs) (Metrics, error)

// CreateLogsToTracesFunc is the equivalent of Factory.CreateLogsToTraces().
type CreateLogsToTracesFunc func(context.Context, Settings, component.Config, consumer.Traces) (Logs, error)

// CreateLogsToMetricsFunc is the equivalent of Factory.CreateLogsToMetrics().
type CreateLogsToMetricsFunc func(context.Context, Settings, component.Config, consumer.Metrics) (Logs, error)

// CreateLogsToLogsFunc is the equivalent of Factory.CreateLogsToLogs().
type CreateLogsToLogsFunc func(context.Context, Settings, component.Config, consumer.Logs) (Logs, error)

// WithTracesToTraces overrides the default "error not supported" implementation for WithTracesToTraces and the default "undefined" stability level.
func WithTracesToTraces(createTracesToTraces CreateTracesToTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.tracesToTracesStabilityLevel = sl
		o.createTracesToTracesFunc = createTracesToTraces
	})
}

// WithTracesToMetrics overrides the default "error not supported" implementation for WithTracesToMetrics and the default "undefined" stability level.
func WithTracesToMetrics(createTracesToMetrics CreateTracesToMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.tracesToMetricsStabilityLevel = sl
		o.createTracesToMetricsFunc = createTracesToMetrics
	})
}

// WithTracesToLogs overrides the default "error not supported" implementation for WithTracesToLogs and the default "undefined" stability level.
func WithTracesToLogs(createTracesToLogs CreateTracesToLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.tracesToLogsStabilityLevel = sl
		o.createTracesToLogsFunc = createTracesToLogs
	})
}

// WithMetricsToTraces overrides the default "error not supported" implementation for WithMetricsToTraces and the default "undefined" stability level.
func WithMetricsToTraces(createMetricsToTraces CreateMetricsToTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.metricsToTracesStabilityLevel = sl
		o.createMetricsToTracesFunc = createMetricsToTraces
	})
}

// WithMetricsToMetrics overrides the default "error not supported" implementation for WithMetricsToMetrics and the default "undefined" stability level.
func WithMetricsToMetrics(createMetricsToMetrics CreateMetricsToMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.metricsToMetricsStabilityLevel = sl
		o.createMetricsToMetricsFunc = createMetricsToMetrics
	})
}

// WithMetricsToLogs overrides the default "error not supported" implementation for WithMetricsToLogs and the default "undefined" stability level.
func WithMetricsToLogs(createMetricsToLogs CreateMetricsToLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.metricsToLogsStabilityLevel = sl
		o.createMetricsToLogsFunc = createMetricsToLogs
	})
}

// WithLogsToTraces overrides the default "error not supported" implementation for WithLogsToTraces and the default "undefined" stability level.
func WithLogsToTraces(createLogsToTraces CreateLogsToTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.logsToTracesStabilityLevel = sl
		o.createLogsToTracesFunc = createLogsToTraces
	})
}

// WithLogsToMetrics overrides the default "error not supported" implementation for WithLogsToMetrics and the default "undefined" stability level.
func WithLogsToMetrics(createLogsToMetrics CreateLogsToMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.logsToMetricsStabilityLevel = sl
		o.createLogsToMetricsFunc = createLogsToMetrics
	})
}

// WithLogsToLogs overrides the default "error not supported" implementation for WithLogsToLogs and the default "undefined" stability level.
func WithLogsToLogs(createLogsToLogs CreateLogsToLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.logsToLogsStabilityLevel = sl
		o.createLogsToLogsFunc = createLogsToLogs
	})
}

// factory implements the Factory interface.
type factory struct {
	cfgType component.Type
	component.CreateDefaultConfigFunc

	createTracesToTracesFunc  CreateTracesToTracesFunc
	createTracesToMetricsFunc CreateTracesToMetricsFunc
	createTracesToLogsFunc    CreateTracesToLogsFunc

	createMetricsToTracesFunc  CreateMetricsToTracesFunc
	createMetricsToMetricsFunc CreateMetricsToMetricsFunc
	createMetricsToLogsFunc    CreateMetricsToLogsFunc

	createLogsToTracesFunc  CreateLogsToTracesFunc
	createLogsToMetricsFunc CreateLogsToMetricsFunc
	createLogsToLogsFunc    CreateLogsToLogsFunc

	tracesToTracesStabilityLevel  component.StabilityLevel
	tracesToMetricsStabilityLevel component.StabilityLevel
	tracesToLogsStabilityLevel    component.StabilityLevel

	metricsToTracesStabilityLevel  component.StabilityLevel
	metricsToMetricsStabilityLevel component.StabilityLevel
	metricsToLogsStabilityLevel    component.StabilityLevel

	logsToTracesStabilityLevel  component.StabilityLevel
	logsToMetricsStabilityLevel component.StabilityLevel
	logsToLogsStabilityLevel    component.StabilityLevel
}

// Type returns the type of component.
func (f *factory) Type() component.Type {
	return f.cfgType
}

func (f *factory) unexportedFactoryFunc() {}

func (f *factory) TracesToTracesStability() component.StabilityLevel {
	return f.tracesToTracesStabilityLevel
}

func (f *factory) TracesToMetricsStability() component.StabilityLevel {
	return f.tracesToMetricsStabilityLevel
}

func (f *factory) TracesToLogsStability() component.StabilityLevel {
	return f.tracesToLogsStabilityLevel
}

func (f *factory) MetricsToTracesStability() component.StabilityLevel {
	return f.metricsToTracesStabilityLevel
}

func (f *factory) MetricsToMetricsStability() component.StabilityLevel {
	return f.metricsToMetricsStabilityLevel
}

func (f *factory) MetricsToLogsStability() component.StabilityLevel {
	return f.metricsToLogsStabilityLevel
}

func (f *factory) LogsToTracesStability() component.StabilityLevel {
	return f.logsToTracesStabilityLevel
}

func (f *factory) LogsToMetricsStability() component.StabilityLevel {
	return f.logsToMetricsStabilityLevel
}

func (f *factory) LogsToLogsStability() component.StabilityLevel {
	return f.logsToLogsStabilityLevel
}

func (f *factory) CreateTracesToTraces(ctx context.Context, set Settings, cfg component.Config, next consumer.Traces) (Traces, error) {
	if f.createTracesToTracesFunc == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalTraces, pipeline.SignalTraces)
	}

	if set.ID.Type() != f.Type() {
		return nil, internal.ErrIDMismatch(set.ID, f.Type())
	}

	return f.createTracesToTracesFunc(ctx, set, cfg, next)
}

func (f *factory) CreateTracesToMetrics(ctx context.Context, set Settings, cfg component.Config, next consumer.Metrics) (Traces, error) {
	if f.createTracesToMetricsFunc == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalTraces, pipeline.SignalMetrics)
	}

	if set.ID.Type() != f.Type() {
		return nil, internal.ErrIDMismatch(set.ID, f.Type())
	}

	return f.createTracesToMetricsFunc(ctx, set, cfg, next)
}

func (f *factory) CreateTracesToLogs(ctx context.Context, set Settings, cfg component.Config, next consumer.Logs) (Traces, error) {
	if f.createTracesToLogsFunc == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalTraces, pipeline.SignalLogs)
	}

	if set.ID.Type() != f.Type() {
		return nil, internal.ErrIDMismatch(set.ID, f.Type())
	}

	return f.createTracesToLogsFunc(ctx, set, cfg, next)
}

func (f *factory) CreateMetricsToTraces(ctx context.Context, set Settings, cfg component.Config, next consumer.Traces) (Metrics, error) {
	if f.createMetricsToTracesFunc == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalMetrics, pipeline.SignalTraces)
	}

	if set.ID.Type() != f.Type() {
		return nil, internal.ErrIDMismatch(set.ID, f.Type())
	}

	return f.createMetricsToTracesFunc(ctx, set, cfg, next)
}

func (f *factory) CreateMetricsToMetrics(ctx context.Context, set Settings, cfg component.Config, next consumer.Metrics) (Metrics, error) {
	if f.createMetricsToMetricsFunc == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalMetrics, pipeline.SignalMetrics)
	}

	if set.ID.Type() != f.Type() {
		return nil, internal.ErrIDMismatch(set.ID, f.Type())
	}

	return f.createMetricsToMetricsFunc(ctx, set, cfg, next)
}

func (f *factory) CreateMetricsToLogs(ctx context.Context, set Settings, cfg component.Config, next consumer.Logs) (Metrics, error) {
	if f.createMetricsToLogsFunc == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalMetrics, pipeline.SignalLogs)
	}

	if set.ID.Type() != f.Type() {
		return nil, internal.ErrIDMismatch(set.ID, f.Type())
	}

	return f.createMetricsToLogsFunc(ctx, set, cfg, next)
}

func (f *factory) CreateLogsToTraces(ctx context.Context, set Settings, cfg component.Config, next consumer.Traces) (Logs, error) {
	if f.createLogsToTracesFunc == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalLogs, pipeline.SignalTraces)
	}

	if set.ID.Type() != f.Type() {
		return nil, internal.ErrIDMismatch(set.ID, f.Type())
	}

	return f.createLogsToTracesFunc(ctx, set, cfg, next)
}

func (f *factory) CreateLogsToMetrics(ctx context.Context, set Settings, cfg component.Config, next consumer.Metrics) (Logs, error) {
	if f.createLogsToMetricsFunc == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalLogs, pipeline.SignalMetrics)
	}

	if set.ID.Type() != f.Type() {
		return nil, internal.ErrIDMismatch(set.ID, f.Type())
	}

	return f.createLogsToMetricsFunc(ctx, set, cfg, next)
}

func (f *factory) CreateLogsToLogs(ctx context.Context, set Settings, cfg component.Config, next consumer.Logs) (Logs, error) {
	if f.createLogsToLogsFunc == nil {
		return nil, internal.ErrDataTypes(set.ID, pipeline.SignalLogs, pipeline.SignalLogs)
	}

	if set.ID.Type() != f.Type() {
		return nil, internal.ErrIDMismatch(set.ID, f.Type())
	}

	return f.createLogsToLogsFunc(ctx, set, cfg, next)
}

// NewFactory returns a Factory.
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	f := &factory{
		cfgType:                 cfgType,
		CreateDefaultConfigFunc: createDefaultConfig,
	}
	for _, opt := range options {
		opt.apply(f)
	}
	return f
}
