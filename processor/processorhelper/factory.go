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

package processorhelper

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
)

// FactoryOption apply changes to ProcessorOptions.
type FactoryOption func(o *factory)

// CreateDefaultConfig is the equivalent of component.ProcessorFactory.CreateDefaultConfig()
type CreateDefaultConfig func() config.Processor

// CreateTracesProcessor is the equivalent of component.ProcessorFactory.CreateTracesProcessor()
type CreateTracesProcessor func(context.Context, component.ProcessorCreateSettings, config.Processor, consumer.Traces) (component.TracesProcessor, error)

// CreateMetricsProcessor is the equivalent of component.ProcessorFactory.CreateMetricsProcessor()
type CreateMetricsProcessor func(context.Context, component.ProcessorCreateSettings, config.Processor, consumer.Metrics) (component.MetricsProcessor, error)

// CreateLogsProcessor is the equivalent of component.ProcessorFactory.CreateLogsProcessor()
type CreateLogsProcessor func(context.Context, component.ProcessorCreateSettings, config.Processor, consumer.Logs) (component.LogsProcessor, error)

type factory struct {
	component.BaseProcessorFactory
	cfgType                config.Type
	createDefaultConfig    CreateDefaultConfig
	createTracesProcessor  CreateTracesProcessor
	createMetricsProcessor CreateMetricsProcessor
	createLogsProcessor    CreateLogsProcessor
}

// WithTraces overrides the default "error not supported" implementation for CreateTracesProcessor.
func WithTraces(createTracesProcessor CreateTracesProcessor) FactoryOption {
	return func(o *factory) {
		o.createTracesProcessor = createTracesProcessor
	}
}

// WithMetrics overrides the default "error not supported" implementation for CreateMetricsProcessor.
func WithMetrics(createMetricsProcessor CreateMetricsProcessor) FactoryOption {
	return func(o *factory) {
		o.createMetricsProcessor = createMetricsProcessor
	}
}

// WithLogs overrides the default "error not supported" implementation for CreateLogsProcessor.
func WithLogs(createLogsProcessor CreateLogsProcessor) FactoryOption {
	return func(o *factory) {
		o.createLogsProcessor = createLogsProcessor
	}
}

// NewFactory returns a component.ProcessorFactory.
func NewFactory(
	cfgType config.Type,
	createDefaultConfig CreateDefaultConfig,
	options ...FactoryOption) component.ProcessorFactory {
	f := &factory{
		cfgType:             cfgType,
		createDefaultConfig: createDefaultConfig,
	}
	for _, opt := range options {
		opt(f)
	}
	return f
}

// Type gets the type of the Processor config created by this factory.
func (f *factory) Type() config.Type {
	return f.cfgType
}

// CreateDefaultConfig creates the default configuration for processor.
func (f *factory) CreateDefaultConfig() config.Processor {
	return f.createDefaultConfig()
}

// CreateTracesProcessor creates a component.TracesProcessor based on this config.
func (f *factory) CreateTracesProcessor(
	ctx context.Context,
	set component.ProcessorCreateSettings,
	cfg config.Processor,
	nextConsumer consumer.Traces,
) (component.TracesProcessor, error) {
	if f.createTracesProcessor == nil {
		return f.BaseProcessorFactory.CreateTracesProcessor(ctx, set, cfg, nextConsumer)
	}
	return f.createTracesProcessor(ctx, set, cfg, nextConsumer)
}

// CreateMetricsProcessor creates a component.MetricsProcessor based on this config.
func (f *factory) CreateMetricsProcessor(
	ctx context.Context,
	set component.ProcessorCreateSettings,
	cfg config.Processor,
	nextConsumer consumer.Metrics,
) (component.MetricsProcessor, error) {
	if f.createMetricsProcessor == nil {
		return f.BaseProcessorFactory.CreateMetricsProcessor(ctx, set, cfg, nextConsumer)
	}
	return f.createMetricsProcessor(ctx, set, cfg, nextConsumer)
}

// CreateLogsProcessor creates a component.LogsProcessor based on this config.
func (f *factory) CreateLogsProcessor(
	ctx context.Context,
	set component.ProcessorCreateSettings,
	cfg config.Processor,
	nextConsumer consumer.Logs,
) (component.LogsProcessor, error) {
	if f.createLogsProcessor == nil {
		return f.BaseProcessorFactory.CreateLogsProcessor(ctx, set, cfg, nextConsumer)
	}
	return f.createLogsProcessor(ctx, set, cfg, nextConsumer)
}
