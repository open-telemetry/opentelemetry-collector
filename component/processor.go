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

package component // import "go.opentelemetry.io/collector/component"

import (
	"context"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/consumer"
)

// ProcessorConfig is the configuration of a component.Processor. Specific Processor must implement
// this interface and must embed ProcessorSettings struct or a struct that extends it.
type ProcessorConfig interface {
	Config
}

// UnmarshalProcessorConfig helper function to unmarshal a ProcessorConfig.
// It checks if the config implements confmap.Unmarshaler and uses that if available,
// otherwise uses Map.UnmarshalExact, erroring if a field is nonexistent.
func UnmarshalProcessorConfig(conf *confmap.Conf, cfg ProcessorConfig) error {
	return unmarshal(conf, cfg)
}

// Deprecated: [v0.65.0] unnecessary interface, will be removed.
type Processor = Component

// TracesProcessor is a processor that can consume traces.
type TracesProcessor interface {
	Component
	consumer.Traces
}

// MetricsProcessor is a processor that can consume metrics.
type MetricsProcessor interface {
	Component
	consumer.Metrics
}

// LogsProcessor is a processor that can consume logs.
type LogsProcessor interface {
	Component
	consumer.Logs
}

// ProcessorCreateSettings is passed to Create* functions in ProcessorFactory.
type ProcessorCreateSettings struct {
	TelemetrySettings

	// BuildInfo can be used by components for informational purposes
	BuildInfo BuildInfo
}

// ProcessorFactory is Factory interface for processors.
//
// This interface cannot be directly implemented. Implementations must
// use the NewProcessorFactory to implement it.
type ProcessorFactory interface {
	Factory

	// CreateDefaultConfig creates the default configuration for the Processor.
	// This method can be called multiple times depending on the pipeline
	// configuration and should not cause side-effects that prevent the creation
	// of multiple instances of the Processor.
	// The object returned by this method needs to pass the checks implemented by
	// 'componenttest.CheckConfigStruct'. It is recommended to have these checks in the
	// tests of any implementation of the Factory interface.
	CreateDefaultConfig() ProcessorConfig

	// CreateTracesProcessor creates a TracesProcessor based on this config.
	// If the processor type does not support tracing or if the config is not valid,
	// an error will be returned instead.
	CreateTracesProcessor(ctx context.Context, set ProcessorCreateSettings, cfg ProcessorConfig, nextConsumer consumer.Traces) (TracesProcessor, error)

	// TracesProcessorStability gets the stability level of the TracesProcessor.
	TracesProcessorStability() StabilityLevel

	// CreateMetricsProcessor creates a MetricsProcessor based on this config.
	// If the processor type does not support metrics or if the config is not valid,
	// an error will be returned instead.
	CreateMetricsProcessor(ctx context.Context, set ProcessorCreateSettings, cfg ProcessorConfig, nextConsumer consumer.Metrics) (MetricsProcessor, error)

	// MetricsProcessorStability gets the stability level of the MetricsProcessor.
	MetricsProcessorStability() StabilityLevel

	// CreateLogsProcessor creates a LogsProcessor based on the config.
	// If the processor type does not support logs or if the config is not valid,
	// an error will be returned instead.
	CreateLogsProcessor(ctx context.Context, set ProcessorCreateSettings, cfg ProcessorConfig, nextConsumer consumer.Logs) (LogsProcessor, error)

	// LogsProcessorStability gets the stability level of the LogsProcessor.
	LogsProcessorStability() StabilityLevel
}

// ProcessorCreateDefaultConfigFunc is the equivalent of ProcessorFactory.CreateDefaultConfig().
type ProcessorCreateDefaultConfigFunc func() ProcessorConfig

// CreateDefaultConfig implements ProcessorFactory.CreateDefaultConfig().
func (f ProcessorCreateDefaultConfigFunc) CreateDefaultConfig() ProcessorConfig {
	return f()
}

// ProcessorFactoryOption apply changes to ProcessorOptions.
type ProcessorFactoryOption interface {
	// applyProcessorFactoryOption applies the option.
	applyProcessorFactoryOption(o *processorFactory)
}

var _ ProcessorFactoryOption = (*processorFactoryOptionFunc)(nil)

// processorFactoryOptionFunc is an ProcessorFactoryOption created through a function.
type processorFactoryOptionFunc func(*processorFactory)

func (f processorFactoryOptionFunc) applyProcessorFactoryOption(o *processorFactory) {
	f(o)
}

// CreateTracesProcessorFunc is the equivalent of ProcessorFactory.CreateTracesProcessor().
type CreateTracesProcessorFunc func(context.Context, ProcessorCreateSettings, ProcessorConfig, consumer.Traces) (TracesProcessor, error)

// CreateTracesProcessor implements ProcessorFactory.CreateTracesProcessor().
func (f CreateTracesProcessorFunc) CreateTracesProcessor(
	ctx context.Context,
	set ProcessorCreateSettings,
	cfg ProcessorConfig,
	nextConsumer consumer.Traces) (TracesProcessor, error) {
	if f == nil {
		return nil, ErrDataTypeIsNotSupported
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateMetricsProcessorFunc is the equivalent of ProcessorFactory.CreateMetricsProcessor().
type CreateMetricsProcessorFunc func(context.Context, ProcessorCreateSettings, ProcessorConfig, consumer.Metrics) (MetricsProcessor, error)

// CreateMetricsProcessor implements ProcessorFactory.CreateMetricsProcessor().
func (f CreateMetricsProcessorFunc) CreateMetricsProcessor(
	ctx context.Context,
	set ProcessorCreateSettings,
	cfg ProcessorConfig,
	nextConsumer consumer.Metrics,
) (MetricsProcessor, error) {
	if f == nil {
		return nil, ErrDataTypeIsNotSupported
	}
	return f(ctx, set, cfg, nextConsumer)
}

// CreateLogsProcessorFunc is the equivalent of ProcessorFactory.CreateLogsProcessor().
type CreateLogsProcessorFunc func(context.Context, ProcessorCreateSettings, ProcessorConfig, consumer.Logs) (LogsProcessor, error)

// CreateLogsProcessor implements ProcessorFactory.CreateLogsProcessor().
func (f CreateLogsProcessorFunc) CreateLogsProcessor(
	ctx context.Context,
	set ProcessorCreateSettings,
	cfg ProcessorConfig,
	nextConsumer consumer.Logs,
) (LogsProcessor, error) {
	if f == nil {
		return nil, ErrDataTypeIsNotSupported
	}
	return f(ctx, set, cfg, nextConsumer)
}

type processorFactory struct {
	baseFactory
	ProcessorCreateDefaultConfigFunc
	CreateTracesProcessorFunc
	tracesStabilityLevel StabilityLevel
	CreateMetricsProcessorFunc
	metricsStabilityLevel StabilityLevel
	CreateLogsProcessorFunc
	logsStabilityLevel StabilityLevel
}

func (p processorFactory) TracesProcessorStability() StabilityLevel {
	return p.tracesStabilityLevel
}

func (p processorFactory) MetricsProcessorStability() StabilityLevel {
	return p.metricsStabilityLevel
}

func (p processorFactory) LogsProcessorStability() StabilityLevel {
	return p.logsStabilityLevel
}

// WithTracesProcessor overrides the default "error not supported" implementation for CreateTracesProcessor and the default "undefined" stability level.
func WithTracesProcessor(createTracesProcessor CreateTracesProcessorFunc, sl StabilityLevel) ProcessorFactoryOption {
	return processorFactoryOptionFunc(func(o *processorFactory) {
		o.tracesStabilityLevel = sl
		o.CreateTracesProcessorFunc = createTracesProcessor
	})
}

// WithMetricsProcessor overrides the default "error not supported" implementation for CreateMetricsProcessor and the default "undefined" stability level.
func WithMetricsProcessor(createMetricsProcessor CreateMetricsProcessorFunc, sl StabilityLevel) ProcessorFactoryOption {
	return processorFactoryOptionFunc(func(o *processorFactory) {
		o.metricsStabilityLevel = sl
		o.CreateMetricsProcessorFunc = createMetricsProcessor
	})
}

// WithLogsProcessor overrides the default "error not supported" implementation for CreateLogsProcessor and the default "undefined" stability level.
func WithLogsProcessor(createLogsProcessor CreateLogsProcessorFunc, sl StabilityLevel) ProcessorFactoryOption {
	return processorFactoryOptionFunc(func(o *processorFactory) {
		o.logsStabilityLevel = sl
		o.CreateLogsProcessorFunc = createLogsProcessor
	})
}

// NewProcessorFactory returns a ProcessorFactory.
func NewProcessorFactory(cfgType Type, createDefaultConfig ProcessorCreateDefaultConfigFunc, options ...ProcessorFactoryOption) ProcessorFactory {
	f := &processorFactory{
		baseFactory:                      baseFactory{cfgType: cfgType},
		ProcessorCreateDefaultConfigFunc: createDefaultConfig,
	}
	for _, opt := range options {
		opt.applyProcessorFactoryOption(f)
	}
	return f
}
