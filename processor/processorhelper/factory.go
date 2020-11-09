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

	"github.com/spf13/viper"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configerror"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
)

// FactoryOption apply changes to ProcessorOptions.
type FactoryOption func(o *factory)

// CreateDefaultConfig is the equivalent of component.ProcessorFactory.CreateDefaultConfig()
type CreateDefaultConfig func() configmodels.Processor

// CreateTraceProcessor is the equivalent of component.ProcessorFactory.CreateTracesProcessor()
type CreateTraceProcessor func(context.Context, component.ProcessorCreateParams, configmodels.Processor, consumer.TracesConsumer) (component.TracesProcessor, error)

// CreateMetricsProcessor is the equivalent of component.ProcessorFactory.CreateMetricsProcessor()
type CreateMetricsProcessor func(context.Context, component.ProcessorCreateParams, configmodels.Processor, consumer.MetricsConsumer) (component.MetricsProcessor, error)

// CreateLogsProcessor is the equivalent of component.ProcessorFactory.CreateLogsProcessor()
type CreateLogsProcessor func(context.Context, component.ProcessorCreateParams, configmodels.Processor, consumer.LogsConsumer) (component.LogsProcessor, error)

type factory struct {
	cfgType                configmodels.Type
	customUnmarshaler      component.CustomUnmarshaler
	createDefaultConfig    CreateDefaultConfig
	createTraceProcessor   CreateTraceProcessor
	createMetricsProcessor CreateMetricsProcessor
	createLogsProcessor    CreateLogsProcessor
}

// WithCustomUnmarshaler implements component.ConfigUnmarshaler.
func WithCustomUnmarshaler(customUnmarshaler component.CustomUnmarshaler) FactoryOption {
	return func(o *factory) {
		o.customUnmarshaler = customUnmarshaler
	}
}

// WithTraces overrides the default "error not supported" implementation for CreateTraceProcessor.
func WithTraces(createTraceProcessor CreateTraceProcessor) FactoryOption {
	return func(o *factory) {
		o.createTraceProcessor = createTraceProcessor
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
	cfgType configmodels.Type,
	createDefaultConfig CreateDefaultConfig,
	options ...FactoryOption) component.ProcessorFactory {
	f := &factory{
		cfgType:             cfgType,
		createDefaultConfig: createDefaultConfig,
	}
	for _, opt := range options {
		opt(f)
	}
	var ret component.ProcessorFactory
	if f.customUnmarshaler != nil {
		ret = &factoryWithUnmarshaler{f}
	} else {
		ret = f
	}
	return ret
}

// Type gets the type of the Processor config created by this factory.
func (f *factory) Type() configmodels.Type {
	return f.cfgType
}

// CreateDefaultConfig creates the default configuration for processor.
func (f *factory) CreateDefaultConfig() configmodels.Processor {
	return f.createDefaultConfig()
}

// CreateTraceProcessor creates a component.TracesProcessor based on this config.
func (f *factory) CreateTracesProcessor(ctx context.Context, params component.ProcessorCreateParams, cfg configmodels.Processor, nextConsumer consumer.TracesConsumer) (component.TracesProcessor, error) {
	if f.createTraceProcessor != nil {
		return f.createTraceProcessor(ctx, params, cfg, nextConsumer)
	}
	return nil, configerror.ErrDataTypeIsNotSupported
}

// CreateMetricsProcessor creates a consumer.MetricsConsumer based on this config.
func (f *factory) CreateMetricsProcessor(ctx context.Context, params component.ProcessorCreateParams, cfg configmodels.Processor, nextConsumer consumer.MetricsConsumer) (component.MetricsProcessor, error) {
	if f.createMetricsProcessor != nil {
		return f.createMetricsProcessor(ctx, params, cfg, nextConsumer)
	}
	return nil, configerror.ErrDataTypeIsNotSupported
}

// CreateLogsProcessor creates a metrics processor based on this config.
func (f *factory) CreateLogsProcessor(
	ctx context.Context,
	params component.ProcessorCreateParams,
	cfg configmodels.Processor,
	nextConsumer consumer.LogsConsumer,
) (component.LogsProcessor, error) {
	if f.createLogsProcessor != nil {
		return f.createLogsProcessor(ctx, params, cfg, nextConsumer)
	}
	return nil, configerror.ErrDataTypeIsNotSupported
}

var _ component.ConfigUnmarshaler = (*factoryWithUnmarshaler)(nil)

type factoryWithUnmarshaler struct {
	*factory
}

// Unmarshal un-marshals the config using the provided custom unmarshaler.
func (f *factoryWithUnmarshaler) Unmarshal(componentViperSection *viper.Viper, intoCfg interface{}) error {
	return f.customUnmarshaler(componentViperSection, intoCfg)
}
