// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
)

var (
	errDataTypesNotSupported = "connection from %s to %s is not supported"
	errTracesToTraces        = fmt.Errorf(errDataTypesNotSupported, component.DataTypeTraces, component.DataTypeTraces)
	errTracesToMetrics       = fmt.Errorf(errDataTypesNotSupported, component.DataTypeTraces, component.DataTypeMetrics)
	errTracesToLogs          = fmt.Errorf(errDataTypesNotSupported, component.DataTypeTraces, component.DataTypeLogs)
	errMetricsToTraces       = fmt.Errorf(errDataTypesNotSupported, component.DataTypeMetrics, component.DataTypeTraces)
	errMetricsToMetrics      = fmt.Errorf(errDataTypesNotSupported, component.DataTypeMetrics, component.DataTypeMetrics)
	errMetricsToLogs         = fmt.Errorf(errDataTypesNotSupported, component.DataTypeMetrics, component.DataTypeLogs)
	errLogsToTraces          = fmt.Errorf(errDataTypesNotSupported, component.DataTypeLogs, component.DataTypeTraces)
	errLogsToMetrics         = fmt.Errorf(errDataTypesNotSupported, component.DataTypeLogs, component.DataTypeMetrics)
	errLogsToLogs            = fmt.Errorf(errDataTypesNotSupported, component.DataTypeLogs, component.DataTypeLogs)
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

// CreateSettings configures Connector creators.
type CreateSettings struct {
	TelemetrySettings component.TelemetrySettings

	// BuildInfo can be used by components for informational purposes
	BuildInfo component.BuildInfo
}

// Factory is factory interface for connectors.
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

	CreateTracesToTraces(ctx context.Context, set CreateSettings, cfg component.Config, nextConsumer consumer.Traces) (Traces, error)
	CreateTracesToMetrics(ctx context.Context, set CreateSettings, cfg component.Config, nextConsumer consumer.Metrics) (Traces, error)
	CreateTracesToLogs(ctx context.Context, set CreateSettings, cfg component.Config, nextConsumer consumer.Logs) (Traces, error)

	CreateMetricsToTraces(ctx context.Context, set CreateSettings, cfg component.Config, nextConsumer consumer.Traces) (Metrics, error)
	CreateMetricsToMetrics(ctx context.Context, set CreateSettings, cfg component.Config, nextConsumer consumer.Metrics) (Metrics, error)
	CreateMetricsToLogs(ctx context.Context, set CreateSettings, cfg component.Config, nextConsumer consumer.Logs) (Metrics, error)

	CreateLogsToTraces(ctx context.Context, set CreateSettings, cfg component.Config, nextConsumer consumer.Traces) (Logs, error)
	CreateLogsToMetrics(ctx context.Context, set CreateSettings, cfg component.Config, nextConsumer consumer.Metrics) (Logs, error)
	CreateLogsToLogs(ctx context.Context, set CreateSettings, cfg component.Config, nextConsumer consumer.Logs) (Logs, error)

	TracesToTracesStability() component.StabilityLevel
	TracesToMetricsStability() component.StabilityLevel
	TracesToLogsStability() component.StabilityLevel

	MetricsToTracesStability() component.StabilityLevel
	MetricsToMetricsStability() component.StabilityLevel
	MetricsToLogsStability() component.StabilityLevel

	LogsToTracesStability() component.StabilityLevel
	LogsToMetricsStability() component.StabilityLevel
	LogsToLogsStability() component.StabilityLevel
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
type CreateTracesToTracesFunc func(context.Context, CreateSettings, component.Config, consumer.Traces) (Traces, error)

// CreateTracesToTraces implements Factory.CreateTracesToTraces().
func (f CreateTracesToTracesFunc) CreateTracesToTraces(
	ctx context.Context,
	set CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces) (Traces, error) {
	if f == nil {
		return nil, errTracesToTraces
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateTracesToMetricsFunc is the equivalent of Factory.CreateTracesToMetrics().
type CreateTracesToMetricsFunc func(context.Context, CreateSettings, component.Config, consumer.Metrics) (Traces, error)

// CreateTracesToMetrics implements Factory.CreateTracesToMetrics().
func (f CreateTracesToMetricsFunc) CreateTracesToMetrics(
	ctx context.Context,
	set CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (Traces, error) {
	if f == nil {
		return nil, errTracesToMetrics
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateTracesToLogsFunc is the equivalent of Factory.CreateTracesToLogs().
type CreateTracesToLogsFunc func(context.Context, CreateSettings, component.Config, consumer.Logs) (Traces, error)

// CreateTracesToLogs implements Factory.CreateTracesToLogs().
func (f CreateTracesToLogsFunc) CreateTracesToLogs(
	ctx context.Context,
	set CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (Traces, error) {
	if f == nil {
		return nil, errTracesToLogs
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateMetricsToTracesFunc is the equivalent of Factory.CreateMetricsToTraces().
type CreateMetricsToTracesFunc func(context.Context, CreateSettings, component.Config, consumer.Traces) (Metrics, error)

// CreateMetricsToTraces implements Factory.CreateMetricsToTraces().
func (f CreateMetricsToTracesFunc) CreateMetricsToTraces(
	ctx context.Context,
	set CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (Metrics, error) {
	if f == nil {
		return nil, errMetricsToTraces
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateMetricsToMetricsFunc is the equivalent of Factory.CreateMetricsToTraces().
type CreateMetricsToMetricsFunc func(context.Context, CreateSettings, component.Config, consumer.Metrics) (Metrics, error)

// CreateMetricsToTraces implements Factory.CreateMetricsToTraces().
func (f CreateMetricsToMetricsFunc) CreateMetricsToMetrics(
	ctx context.Context,
	set CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (Metrics, error) {
	if f == nil {
		return nil, errMetricsToMetrics
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateMetricsToLogsFunc is the equivalent of Factory.CreateMetricsToLogs().
type CreateMetricsToLogsFunc func(context.Context, CreateSettings, component.Config, consumer.Logs) (Metrics, error)

// CreateMetricsToLogs implements Factory.CreateMetricsToLogs().
func (f CreateMetricsToLogsFunc) CreateMetricsToLogs(
	ctx context.Context,
	set CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (Metrics, error) {
	if f == nil {
		return nil, errMetricsToLogs
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateLogsToTracesFunc is the equivalent of Factory.CreateLogsToTraces().
type CreateLogsToTracesFunc func(context.Context, CreateSettings, component.Config, consumer.Traces) (Logs, error)

// CreateLogsToTraces implements Factory.CreateLogsToTraces().
func (f CreateLogsToTracesFunc) CreateLogsToTraces(
	ctx context.Context,
	set CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (Logs, error) {
	if f == nil {
		return nil, errLogsToTraces
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateLogsToMetricssFunc is the equivalent of Factory.CreateLogsToMetrics().
type CreateLogsToMetricsFunc func(context.Context, CreateSettings, component.Config, consumer.Metrics) (Logs, error)

// CreateLogsToMetrics implements Factory.CreateLogsToMetrics().
func (f CreateLogsToMetricsFunc) CreateLogsToMetrics(
	ctx context.Context,
	set CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (Logs, error) {
	if f == nil {
		return nil, errLogsToMetrics
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateLogsToLogsFunc is the equivalent of Factory.CreateLogsToLogs().
type CreateLogsToLogsFunc func(context.Context, CreateSettings, component.Config, consumer.Logs) (Logs, error)

// CreateLogsToLogs implements Factory.CreateLogsToLogs().
func (f CreateLogsToLogsFunc) CreateLogsToLogs(
	ctx context.Context,
	set CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (Logs, error) {
	if f == nil {
		return nil, errLogsToLogs
	}
	return f(ctx, set, cfg, nextConsumer)
}

// factory implements Factory.
type factory struct {
	component.Factory
	cfgType component.Type
	component.CreateDefaultConfigFunc

	CreateTracesToTracesFunc
	CreateTracesToMetricsFunc
	CreateTracesToLogsFunc

	CreateMetricsToTracesFunc
	CreateMetricsToMetricsFunc
	CreateMetricsToLogsFunc

	CreateLogsToTracesFunc
	CreateLogsToMetricsFunc
	CreateLogsToLogsFunc

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

var _ Factory = (*factory)(nil)

// Type returns the type of component.
func (f *factory) Type() component.Type {
	return f.cfgType
}

// CreateDefaultConfig creates the default configuration for the Component.
// TODO: Remove this when we remove the private func from component.Factory and add it to every specialized Factory.
func (f *factory) CreateDefaultConfig() component.Config {
	return f.CreateDefaultConfigFunc()
}

// WithTracesToTraces overrides the default "error not supported" implementation for WithTracesToTraces and the default "undefined" stability level.
func WithTracesToTraces(createTracesToTraces CreateTracesToTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.tracesToTracesStabilityLevel = sl
		o.CreateTracesToTracesFunc = createTracesToTraces
	})
}

// WithTracesToMetrics overrides the default "error not supported" implementation for WithTracesToMetrics and the default "undefined" stability level.
func WithTracesToMetrics(createTracesToMetrics CreateTracesToMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.tracesToMetricsStabilityLevel = sl
		o.CreateTracesToMetricsFunc = createTracesToMetrics
	})
}

// WithTracesToLogs overrides the default "error not supported" implementation for WithTracesToLogs and the default "undefined" stability level.
func WithTracesToLogs(createTracesToLogs CreateTracesToLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.tracesToLogsStabilityLevel = sl
		o.CreateTracesToLogsFunc = createTracesToLogs
	})
}

// WithMetricsToTraces overrides the default "error not supported" implementation for WithMetricsToTraces and the default "undefined" stability level.
func WithMetricsToTraces(createMetricsToTraces CreateMetricsToTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.metricsToTracesStabilityLevel = sl
		o.CreateMetricsToTracesFunc = createMetricsToTraces
	})
}

// WithMetricsToMetrics overrides the default "error not supported" implementation for WithMetricsToMetrics and the default "undefined" stability level.
func WithMetricsToMetrics(createMetricsToMetrics CreateMetricsToMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.metricsToMetricsStabilityLevel = sl
		o.CreateMetricsToMetricsFunc = createMetricsToMetrics
	})
}

// WithMetricsToLogs overrides the default "error not supported" implementation for WithMetricsToLogs and the default "undefined" stability level.
func WithMetricsToLogs(createMetricsToLogs CreateMetricsToLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.metricsToLogsStabilityLevel = sl
		o.CreateMetricsToLogsFunc = createMetricsToLogs
	})
}

// WithLogsToTraces overrides the default "error not supported" implementation for WithLogsToTraces and the default "undefined" stability level.
func WithLogsToTraces(createLogsToTraces CreateLogsToTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.logsToTracesStabilityLevel = sl
		o.CreateLogsToTracesFunc = createLogsToTraces
	})
}

// WithLogsToMetrics overrides the default "error not supported" implementation for WithLogsToMetrics and the default "undefined" stability level.
func WithLogsToMetrics(createLogsToMetrics CreateLogsToMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.logsToMetricsStabilityLevel = sl
		o.CreateLogsToMetricsFunc = createLogsToMetrics
	})
}

// WithLogsToLogs overrides the default "error not supported" implementation for WithLogsToLogs and the default "undefined" stability level.
func WithLogsToLogs(createLogsToLogs CreateLogsToLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.logsToLogsStabilityLevel = sl
		o.CreateLogsToLogsFunc = createLogsToLogs
	})
}

func (f factory) TracesToTracesStability() component.StabilityLevel {
	return f.tracesToTracesStabilityLevel
}

func (f factory) TracesToMetricsStability() component.StabilityLevel {
	return f.tracesToMetricsStabilityLevel
}

func (f factory) TracesToLogsStability() component.StabilityLevel {
	return f.tracesToLogsStabilityLevel
}

func (f factory) MetricsToTracesStability() component.StabilityLevel {
	return f.metricsToTracesStabilityLevel
}

func (f factory) MetricsToMetricsStability() component.StabilityLevel {
	return f.metricsToMetricsStabilityLevel
}

func (f factory) MetricsToLogsStability() component.StabilityLevel {
	return f.metricsToLogsStabilityLevel
}

func (f factory) LogsToTracesStability() component.StabilityLevel {
	return f.logsToTracesStabilityLevel
}

func (f factory) LogsToMetricsStability() component.StabilityLevel {
	return f.logsToMetricsStabilityLevel
}

func (f factory) LogsToLogsStability() component.StabilityLevel {
	return f.logsToLogsStabilityLevel
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

// MakeFactoryMap takes a list of connector factories and returns a map with factory type as keys.
// It returns a non-nil error when there are factories with duplicate type.
func MakeFactoryMap(factories ...Factory) (map[component.Type]Factory, error) {
	fMap := map[component.Type]Factory{}
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return fMap, fmt.Errorf("duplicate connector factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}
