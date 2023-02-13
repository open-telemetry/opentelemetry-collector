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

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
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

// TracesRouter feeds the first consumer.Traces in each of the specified pipelines.
// The router will create a fanout consumer for the set of pipelines and return a uuid
type TracesRouter interface {
	Consumer(...component.ID) (consumer.Traces, error)
	PipelineIDs() []component.ID
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

// MetricsRouter feeds the first consumer.Metrics in each of the specified pipelines.
type MetricsRouter interface {
	Consumer(...component.ID) (consumer.Metrics, error)
	PipelineIDs() []component.ID
}

// A Logs connector acts as an exporter from a logs pipeline and a receiver
// to one or more traces, metrics, or logs pipelines.
// Logs feeds a consumer.Logs, consumer.Metrics, or consumer.Logs with data.
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

// LogsRouter feeds the first consumer.Logs in each of the specified pipelines.
type LogsRouter interface {
	Consumer(...component.ID) (consumer.Logs, error)
	PipelineIDs() []component.ID
}

// CreateSettings configures Connector creators.
type CreateSettings struct {
	// ID returns the ID of the component that will be created.
	ID component.ID

	component.TelemetrySettings

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
type CreateTracesToTracesFunc func(context.Context, CreateSettings, component.Config, consumer.Traces) (Traces, error)

// CreateTracesToTraces implements Factory.CreateTracesToTraces().
func (f CreateTracesToTracesFunc) CreateTracesToTraces(
	ctx context.Context,
	set CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces) (Traces, error) {
	if f == nil {
		return nil, errDataTypes(set.ID, component.DataTypeTraces, component.DataTypeTraces)
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
		return nil, errDataTypes(set.ID, component.DataTypeTraces, component.DataTypeMetrics)
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
		return nil, errDataTypes(set.ID, component.DataTypeTraces, component.DataTypeLogs)
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
		return nil, errDataTypes(set.ID, component.DataTypeMetrics, component.DataTypeTraces)
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateMetricsToMetricsFunc is the equivalent of Factory.CreateMetricsToTraces().
type CreateMetricsToMetricsFunc func(context.Context, CreateSettings, component.Config, consumer.Metrics) (Metrics, error)

// CreateMetricsToMetrics implements Factory.CreateMetricsToTraces().
func (f CreateMetricsToMetricsFunc) CreateMetricsToMetrics(
	ctx context.Context,
	set CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (Metrics, error) {
	if f == nil {
		return nil, errDataTypes(set.ID, component.DataTypeMetrics, component.DataTypeMetrics)
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
		return nil, errDataTypes(set.ID, component.DataTypeMetrics, component.DataTypeLogs)
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
		return nil, errDataTypes(set.ID, component.DataTypeLogs, component.DataTypeTraces)
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateLogsToMetricsFunc is the equivalent of Factory.CreateLogsToMetrics().
type CreateLogsToMetricsFunc func(context.Context, CreateSettings, component.Config, consumer.Metrics) (Logs, error)

// CreateLogsToMetrics implements Factory.CreateLogsToMetrics().
func (f CreateLogsToMetricsFunc) CreateLogsToMetrics(
	ctx context.Context,
	set CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (Logs, error) {
	if f == nil {
		return nil, errDataTypes(set.ID, component.DataTypeLogs, component.DataTypeMetrics)
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
		return nil, errDataTypes(set.ID, component.DataTypeLogs, component.DataTypeLogs)
	}
	return f(ctx, set, cfg, nextConsumer)
}

// factory implements Factory.
type factory struct {
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

// Type returns the type of component.
func (f *factory) Type() component.Type {
	return f.cfgType
}

func (f *factory) unexportedFactoryFunc() {}

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

// Builder processor is a helper struct that given a set of Configs and Factories helps with creating processors.
type Builder struct {
	cfgs      map[component.ID]component.Config
	factories map[component.Type]Factory
}

// NewBuilder creates a new connector.Builder to help with creating components form a set of configs and factories.
func NewBuilder(cfgs map[component.ID]component.Config, factories map[component.Type]Factory) *Builder {
	return &Builder{cfgs: cfgs, factories: factories}
}

// CreateTracesToTraces creates a Traces connector based on the settings and config.
func (b *Builder) CreateTracesToTraces(ctx context.Context, set CreateSettings, next consumer.Traces) (Traces, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.TracesToTracesStability())
	return f.CreateTracesToTraces(ctx, set, cfg, next)
}

// CreateTracesToMetrics creates a Traces connector based on the settings and config.
func (b *Builder) CreateTracesToMetrics(ctx context.Context, set CreateSettings, next consumer.Metrics) (Traces, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.TracesToMetricsStability())
	return f.CreateTracesToMetrics(ctx, set, cfg, next)
}

// CreateTracesToLogs creates a Traces connector based on the settings and config.
func (b *Builder) CreateTracesToLogs(ctx context.Context, set CreateSettings, next consumer.Logs) (Traces, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.TracesToLogsStability())
	return f.CreateTracesToLogs(ctx, set, cfg, next)
}

// CreateMetricsToTraces creates a Metrics connector based on the settings and config.
func (b *Builder) CreateMetricsToTraces(ctx context.Context, set CreateSettings, next consumer.Traces) (Metrics, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.MetricsToTracesStability())
	return f.CreateMetricsToTraces(ctx, set, cfg, next)
}

// CreateMetricsToMetrics creates a Metrics connector based on the settings and config.
func (b *Builder) CreateMetricsToMetrics(ctx context.Context, set CreateSettings, next consumer.Metrics) (Metrics, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.MetricsToMetricsStability())
	return f.CreateMetricsToMetrics(ctx, set, cfg, next)
}

// CreateMetricsToLogs creates a Metrics connector based on the settings and config.
func (b *Builder) CreateMetricsToLogs(ctx context.Context, set CreateSettings, next consumer.Logs) (Metrics, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.MetricsToLogsStability())
	return f.CreateMetricsToLogs(ctx, set, cfg, next)
}

// CreateLogsToTraces creates a Logs connector based on the settings and config.
func (b *Builder) CreateLogsToTraces(ctx context.Context, set CreateSettings, next consumer.Traces) (Logs, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.LogsToTracesStability())
	return f.CreateLogsToTraces(ctx, set, cfg, next)
}

// CreateLogsToMetrics creates a Logs connector based on the settings and config.
func (b *Builder) CreateLogsToMetrics(ctx context.Context, set CreateSettings, next consumer.Metrics) (Logs, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.LogsToMetricsStability())
	return f.CreateLogsToMetrics(ctx, set, cfg, next)
}

// CreateLogsToLogs creates a Logs connector based on the settings and config.
func (b *Builder) CreateLogsToLogs(ctx context.Context, set CreateSettings, next consumer.Logs) (Logs, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("connector %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("connector factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.LogsToLogsStability())
	return f.CreateLogsToLogs(ctx, set, cfg, next)
}

func (b *Builder) IsConfigured(componentID component.ID) bool {
	_, ok := b.cfgs[componentID]
	return ok
}

func (b *Builder) Factory(componentType component.Type) component.Factory {
	return b.factories[componentType]
}

// logStabilityLevel logs the stability level of a component. The log level is set to info for
// undefined, unmaintained, deprecated and development. The log level is set to debug
// for alpha, beta and stable.
func logStabilityLevel(logger *zap.Logger, sl component.StabilityLevel) {
	if sl >= component.StabilityLevelAlpha {
		logger.Debug(sl.LogMessage())
	} else {
		logger.Info(sl.LogMessage())
	}
}

func errDataTypes(id component.ID, from, to component.DataType) error {
	return fmt.Errorf("connector %q cannot connect from %s to %s: %w", id, from, to, component.ErrDataTypeIsNotSupported)
}
