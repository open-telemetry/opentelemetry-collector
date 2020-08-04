// Copyright  OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package receiverhelper

import (
	"context"

	"github.com/spf13/viper"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configerror"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
)

// FactoryOption apply changes to ReceiverOptions.
type FactoryOption func(o *factory)

// WithCustomUnmarshaler implements component.ConfigUnmarshaler.
func WithCustomUnmarshaler(customUnmarshaler component.CustomUnmarshaler) FactoryOption {
	return func(o *factory) {
		o.customUnmarshaler = customUnmarshaler
	}
}

// WithTraces overrides the default "error not supported" implementation for CreateTraceReceiver.
func WithTraces(createTraceReceiver CreateTraceReceiver) FactoryOption {
	return func(o *factory) {
		o.createTraceReceiver = createTraceReceiver
	}
}

// WithMetrics overrides the default "error not supported" implementation for CreateMetricsReceiver.
func WithMetrics(createMetricsReceiver CreateMetricsReceiver) FactoryOption {
	return func(o *factory) {
		o.createMetricsReceiver = createMetricsReceiver
	}
}

// WithLogs overrides the default "error not supported" implementation for CreateLogsReceiver.
func WithLogs(createLogsReceiver CreateLogsReceiver) FactoryOption {
	return func(o *factory) {
		o.createLogsReceiver = createLogsReceiver
	}
}

// WithSharedTraces overrides the default "error not supported" implementation for CreateTraceReceiver.
func WithSharedTraces(createSharedTraceReceiver CreateSharedTraceReceiver) FactoryOption {
	return func(o *factory) {
		o.createSharedTraceReceiver = createSharedTraceReceiver
	}
}

// WithSharedMetrics overrides the default "error not supported" implementation for createSharedSharedMetricsReceiver.
func WithSharedMetrics(createSharedMetricsReceiver CreateSharedMetricsReceiver) FactoryOption {
	return func(o *factory) {
		o.createSharedMetricsReceiver = createSharedMetricsReceiver
	}
}

// WithSharedLogs overrides the default "error not supported" implementation for createSharedSharedLogsReceiver.
func WithSharedLogs(createSharedLogsReceiver CreateSharedLogsReceiver) FactoryOption {
	return func(o *factory) {
		o.createSharedLogsReceiver = createSharedLogsReceiver
	}
}

// WithSharedPerConfig can be set to a function that will create a generic
// instance of something that will be provided to each Create*Receiver method
// as part of the params.SharedPerConfig object.
func WithSharedPerConfig(f func(configmodels.Receiver, component.ReceiverCreateParams) (interface{}, error)) FactoryOption {
	return func(o *factory) {
		o.sharedPerConfigMaker = f
	}
}

// CreateDefaultConfig is the equivalent of component.ReceiverFactory.CreateDefaultConfig()
type CreateDefaultConfig func() configmodels.Receiver

// CreateTraceReceiver is the equivalent of component.ReceiverFactory.CreateTraceReceiver()
type CreateTraceReceiver func(context.Context, component.ReceiverCreateParams, configmodels.Receiver, consumer.TraceConsumer) (component.TraceReceiver, error)

// CreateMetricsReceiver is the equivalent of component.ReceiverFactory.CreateMetricsReceiver()
type CreateMetricsReceiver func(context.Context, component.ReceiverCreateParams, configmodels.Receiver, consumer.MetricsConsumer) (component.MetricsReceiver, error)

// CreateLogsReceiver is the equivalent of component.ReceiverFactory.CreateLogsReceiver()
type CreateLogsReceiver func(context.Context, component.ReceiverCreateParams, configmodels.Receiver, consumer.LogsConsumer) (component.LogsReceiver, error)

// CreateSharedTraceReceiver is the equivalent of component.ReceiverFactory.CreateTraceReceiver()
type CreateSharedTraceReceiver func(context.Context, component.ReceiverCreateParams, configmodels.Receiver, consumer.TraceConsumer, interface{}) (component.TraceReceiver, error)

// CreateSharedMetricsReceiver is the equivalent of component.ReceiverFactory.CreateMetricsReceiver()
type CreateSharedMetricsReceiver func(context.Context, component.ReceiverCreateParams, configmodels.Receiver, consumer.MetricsConsumer, interface{}) (component.MetricsReceiver, error)

// CreateSharedLogsReceiver is the equivalent of component.ReceiverFactory.CreateLogsReceiver()
type CreateSharedLogsReceiver func(context.Context, component.ReceiverCreateParams, configmodels.Receiver, consumer.LogsConsumer, interface{}) (component.LogsReceiver, error)

type factory struct {
	cfgType                     configmodels.Type
	sharedPerConfigMaker        func(configmodels.Receiver, component.ReceiverCreateParams) (interface{}, error)
	sharedPerConfigInstances    map[configmodels.Receiver]interface{}
	customUnmarshaler           component.CustomUnmarshaler
	createDefaultConfig         CreateDefaultConfig
	createTraceReceiver         CreateTraceReceiver
	createMetricsReceiver       CreateMetricsReceiver
	createLogsReceiver          CreateLogsReceiver
	createSharedTraceReceiver   CreateSharedTraceReceiver
	createSharedMetricsReceiver CreateSharedMetricsReceiver
	createSharedLogsReceiver    CreateSharedLogsReceiver
}

// NewFactory returns a component.ReceiverFactory.
func NewFactory(
	cfgType configmodels.Type,
	createDefaultConfig CreateDefaultConfig,
	options ...FactoryOption) component.ReceiverFactory {
	f := &factory{
		cfgType:                  cfgType,
		createDefaultConfig:      createDefaultConfig,
		sharedPerConfigInstances: make(map[configmodels.Receiver]interface{}),
	}
	for _, opt := range options {
		opt(f)
	}
	var ret component.ReceiverFactory
	if f.customUnmarshaler != nil {
		ret = &factoryWithUnmarshaler{f}
	} else {
		ret = f
	}
	return ret
}

// Type gets the type of the Receiver config created by this factory.
func (f *factory) Type() configmodels.Type {
	return f.cfgType
}

// CreateDefaultConfig creates the default configuration for receiver.
func (f *factory) CreateDefaultConfig() configmodels.Receiver {
	return f.createDefaultConfig()
}

func (f *factory) getSharedPerConfig(cfg configmodels.Receiver, params *component.ReceiverCreateParams) (interface{}, error) {
	if f.sharedPerConfigMaker != nil && f.sharedPerConfigInstances[cfg] == nil {
		inst, err := f.sharedPerConfigMaker(cfg, *params)
		if err != nil {
			return nil, err
		}
		f.sharedPerConfigInstances[cfg] = inst
	}

	return f.sharedPerConfigInstances[cfg], nil
}

// CreateTraceReceiver creates a component.TraceReceiver based on this config.
func (f *factory) CreateTraceReceiver(
	ctx context.Context,
	params component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	nextConsumer consumer.TraceConsumer) (component.TraceReceiver, error) {
	if f.createTraceReceiver != nil {
		return f.createTraceReceiver(ctx, params, cfg, nextConsumer)
	} else if f.createSharedTraceReceiver != nil {
		shared, err := f.getSharedPerConfig(cfg, &params)
		if err != nil {
			return nil, err
		}
		return f.createSharedTraceReceiver(ctx, params, cfg, nextConsumer, shared)
	}
	return nil, configerror.ErrDataTypeIsNotSupported
}

// CreateMetricsReceiver creates a consumer.MetricsConsumer based on this config.
func (f *factory) CreateMetricsReceiver(
	ctx context.Context,
	params component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	nextConsumer consumer.MetricsConsumer) (component.MetricsReceiver, error) {
	if f.createMetricsReceiver != nil {
		return f.createMetricsReceiver(ctx, params, cfg, nextConsumer)
	} else if f.createSharedMetricsReceiver != nil {
		shared, err := f.getSharedPerConfig(cfg, &params)
		if err != nil {
			return nil, err
		}
		return f.createSharedMetricsReceiver(ctx, params, cfg, nextConsumer, shared)
	}
	return nil, configerror.ErrDataTypeIsNotSupported
}

// CreateLogsReceiver creates a metrics processor based on this config.
func (f *factory) CreateLogsReceiver(
	ctx context.Context,
	params component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	nextConsumer consumer.LogsConsumer,
) (component.LogsReceiver, error) {
	if f.createLogsReceiver != nil {
		return f.createLogsReceiver(ctx, params, cfg, nextConsumer)
	} else if f.createSharedLogsReceiver != nil {
		shared, err := f.getSharedPerConfig(cfg, &params)
		if err != nil {
			return nil, err
		}
		return f.createSharedLogsReceiver(ctx, params, cfg, nextConsumer, shared)
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
