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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configerror"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
)

// FactoryOption apply changes to ReceiverOptions.
type FactoryOption func(o *factory)

// WithCustomUnmarshaler overrides the default "not available" CustomUnmarshaler.
func WithCustomUnmarshaler(customUnmarshaler component.CustomUnmarshaler) FactoryOption {
	return func(o *factory) {
		o.customUnmarshaler = customUnmarshaler
	}
}

// WithTraceReceiver overrides the default "error not supported" implementation for CreateTraceReceiver.
func WithTraceReceiver(createTraceReceiver CreateTraceReceiver) FactoryOption {
	return func(o *factory) {
		o.createTraceReceiver = createTraceReceiver
	}
}

// WithTraceReceiver overrides the default "error not supported" implementation for CreateMetricsReceiver.
func WithMetricsReceiver(createMetricsReceiver CreateMetricsReceiver) FactoryOption {
	return func(o *factory) {
		o.createMetricsReceiver = createMetricsReceiver
	}
}

// CreateDefaultConfig is the equivalent of component.ReceiverFactory.CreateDefaultConfig()
type CreateDefaultConfig func() configmodels.Receiver

// CreateTraceReceiver is the equivalent of component.ReceiverFactory.CreateTraceReceiver()
type CreateTraceReceiver func(ctx context.Context, params component.ReceiverCreateParams, config configmodels.Receiver, nextConsumer consumer.TraceConsumer) (component.TraceReceiver, error)

// CreateMetricsReceiver is the equivalent of component.ReceiverFactory.CreateMetricsReceiver()
type CreateMetricsReceiver func(ctx context.Context, params component.ReceiverCreateParams, config configmodels.Receiver, nextConsumer consumer.MetricsConsumer) (component.MetricsReceiver, error)

// factory is the factory for Jaeger gRPC exporter.
type factory struct {
	cfgType               configmodels.Type
	customUnmarshaler     component.CustomUnmarshaler
	createDefaultConfig   CreateDefaultConfig
	createTraceReceiver   CreateTraceReceiver
	createMetricsReceiver CreateMetricsReceiver
}

// NewFactory returns a component.ReceiverFactory that only supports all types.
func NewFactory(
	cfgType configmodels.Type,
	createDefaultConfig CreateDefaultConfig,
	options ...FactoryOption) component.ReceiverFactory {
	f := &factory{
		cfgType:             cfgType,
		createDefaultConfig: createDefaultConfig,
	}
	for _, opt := range options {
		opt(f)
	}
	return f
}

// Type gets the type of the Receiver config created by this factory.
func (f *factory) Type() configmodels.Type {
	return f.cfgType
}

// CustomUnmarshaler returns a custom unmarshaler for the configuration or nil if
// there is no need for custom unmarshaling.
func (f *factory) CustomUnmarshaler() component.CustomUnmarshaler {
	return f.customUnmarshaler
}

// CreateDefaultConfig creates the default configuration for receiver.
func (f *factory) CreateDefaultConfig() configmodels.Receiver {
	return f.createDefaultConfig()
}

// CreateTraceReceiver creates a component.TraceReceiver based on this config.
func (f *factory) CreateTraceReceiver(
	ctx context.Context,
	params component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	nextConsumer consumer.TraceConsumer) (component.TraceReceiver, error) {
	if f.createTraceReceiver != nil {
		return f.createTraceReceiver(ctx, params, cfg, nextConsumer)
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
	}
	return nil, configerror.ErrDataTypeIsNotSupported
}
