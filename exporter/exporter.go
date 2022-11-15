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

package exporter // import "go.opentelemetry.io/collector/component"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
)

// Config is the configuration of any exporter. Specific exporters must implement
// this interface and must embed config.ExporterSettings struct or a struct that extends it.
type Config interface {
	component.Config

	privateConfigExporter()
}

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

// CreateSettings configures exporter creators.
type CreateSettings struct {
	TelemetrySettings component.TelemetrySettings

	// BuildInfo can be used by components for informational purposes
	BuildInfo component.BuildInfo
}

// Factory is factory interface for exporters.
//
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
type Factory interface {
	component.Factory

	// CreateDefaultConfig creates the default configuration for the exporter.
	// This method can be called multiple times depending on the pipeline
	// configuration and should not cause side-effects that prevent the creation
	// of multiple instances of the exporter.
	// The object returned by this method needs to pass the checks implemented by
	// 'componenttest.CheckConfigStruct'. It is recommended to have these checks in the
	// tests of any implementation of the Factory interface.
	CreateDefaultConfig() Config

	// CreateTracesExporter creates an exporter.Traces based on this config.
	// If the exporter type does not support tracing or if the config is not valid,
	// an error will be returned instead.
	CreateTracesExporter(ctx context.Context, set CreateSettings, cfg Config) (Traces, error)

	// TracesExporterStability gets the stability level of the exporter.Traces.
	TracesExporterStability() component.StabilityLevel

	// CreateMetricsExporter creates an exporter.Metrics based on this config.
	// If the exporter type does not support metrics or if the config is not valid,
	// an error will be returned instead.
	CreateMetricsExporter(ctx context.Context, set CreateSettings, cfg Config) (Metrics, error)

	// MetricsExporterStability gets the stability level of the exporter.Metrics.
	MetricsExporterStability() component.StabilityLevel

	// CreateLogsExporter creates an exporter.Logs based on the config.
	// If the exporter type does not support logs or if the config is not valid,
	// an error will be returned instead.
	CreateLogsExporter(ctx context.Context, set CreateSettings, cfg Config) (Logs, error)

	// LogsExporterStability gets the stability level of the exporter.Logs.
	LogsExporterStability() component.StabilityLevel

	unexportedFunc()
}

// FactoryOption apply changes to NewFactory.
type FactoryOption interface {
	// apply the option.
	apply(o *factory)
}

var _ FactoryOption = (*factoryOptionFunc)(nil)

// factoryOptionFunc is an FactoryOption created through a function.
type factoryOptionFunc func(*factory)

func (f factoryOptionFunc) apply(o *factory) {
	f(o)
}

// CreateDefaultConfigFunc is the equivalent of Factory.CreateDefaultConfig().
type CreateDefaultConfigFunc func() Config

// CreateDefaultConfig implements Factory.CreateDefaultConfig().
func (f CreateDefaultConfigFunc) CreateDefaultConfig() Config {
	return f()
}

// CreateTracesFunc is the equivalent of Factory.CreateTracesExporter().
type CreateTracesFunc func(context.Context, CreateSettings, Config) (Traces, error)

// CreateTracesExporter implements Factory.CreateTracesExporter().
func (f CreateTracesFunc) CreateTracesExporter(ctx context.Context, set CreateSettings, cfg Config) (Traces, error) {
	if f == nil {
		return nil, component.ErrDataTypeIsNotSupported
	}
	return f(ctx, set, cfg)
}

// CreateMetricsFunc is the equivalent of Factory.CreateMetricsExporter().
type CreateMetricsFunc func(context.Context, CreateSettings, Config) (Metrics, error)

// CreateMetricsExporter implements Factory.CreateMetricsExporter().
func (f CreateMetricsFunc) CreateMetricsExporter(ctx context.Context, set CreateSettings, cfg Config) (Metrics, error) {
	if f == nil {
		return nil, component.ErrDataTypeIsNotSupported
	}
	return f(ctx, set, cfg)
}

// CreateLogsFunc is the equivalent of Factory.CreateLogsExporter().
type CreateLogsFunc func(context.Context, CreateSettings, Config) (Logs, error)

// CreateLogsExporter implements Factory.CreateLogsExporter().
func (f CreateLogsFunc) CreateLogsExporter(ctx context.Context, set CreateSettings, cfg Config) (Logs, error) {
	if f == nil {
		return nil, component.ErrDataTypeIsNotSupported
	}
	return f(ctx, set, cfg)
}

type factory struct {
	cfgType component.Type
	CreateDefaultConfigFunc
	CreateTracesFunc
	tracesStabilityLevel component.StabilityLevel
	CreateMetricsFunc
	metricsStabilityLevel component.StabilityLevel
	CreateLogsFunc
	logsStabilityLevel component.StabilityLevel
}

func (f factory) Type() component.Type {
	return f.cfgType
}

func (f factory) unexportedFunc() {}

func (f factory) TracesExporterStability() component.StabilityLevel {
	return f.tracesStabilityLevel
}

func (f factory) MetricsExporterStability() component.StabilityLevel {
	return f.metricsStabilityLevel
}

func (f factory) LogsExporterStability() component.StabilityLevel {
	return f.logsStabilityLevel
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

// NewFactory returns a Factory.
func NewFactory(cfgType component.Type, createDefaultConfig CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	f := &factory{
		cfgType:                 cfgType,
		CreateDefaultConfigFunc: createDefaultConfig,
	}
	for _, opt := range options {
		opt.apply(f)
	}
	return f
}
