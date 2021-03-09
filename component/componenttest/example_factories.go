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

package componenttest

import (
	"context"

	"github.com/spf13/viper"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
)

// MultiProtoReceiver is for testing purposes. We are defining an example multi protocol
// config and factory for "multireceiver" receiver type.
type MultiProtoReceiver struct {
	configmodels.ReceiverSettings `mapstructure:",squash"`            // squash ensures fields are correctly decoded in embedded struct
	Protocols                     map[string]MultiProtoReceiverOneCfg `mapstructure:"protocols"`
}

// MultiProtoReceiverOneCfg is multi proto receiver config.
type MultiProtoReceiverOneCfg struct {
	Endpoint     string `mapstructure:"endpoint"`
	ExtraSetting string `mapstructure:"extra"`
}

// MultiProtoReceiverFactory is factory for MultiProtoReceiver.
type MultiProtoReceiverFactory struct {
}

var _ component.ReceiverFactory = (*MultiProtoReceiverFactory)(nil)

// Type gets the type of the Receiver config created by this factory.
func (f *MultiProtoReceiverFactory) Type() configmodels.Type {
	return "multireceiver"
}

// Unmarshal implements the ConfigUnmarshaler interface.
func (f *MultiProtoReceiverFactory) Unmarshal(componentViperSection *viper.Viper, intoCfg interface{}) error {
	return componentViperSection.UnmarshalExact(intoCfg)
}

// CreateDefaultConfig creates the default configuration for the Receiver.
func (f *MultiProtoReceiverFactory) CreateDefaultConfig() configmodels.Receiver {
	return &MultiProtoReceiver{
		ReceiverSettings: configmodels.ReceiverSettings{
			TypeVal: f.Type(),
			NameVal: string(f.Type()),
		},
		Protocols: map[string]MultiProtoReceiverOneCfg{
			"http": {
				Endpoint:     "example.com:8888",
				ExtraSetting: "extra string 1",
			},
			"tcp": {
				Endpoint:     "omnition.com:9999",
				ExtraSetting: "extra string 2",
			},
		},
	}
}

// CreateTraceReceiver creates a trace receiver based on this config.
func (f *MultiProtoReceiverFactory) CreateTracesReceiver(
	_ context.Context,
	_ component.ReceiverCreateParams,
	_ configmodels.Receiver,
	_ consumer.TracesConsumer,
) (component.TracesReceiver, error) {
	// Not used for this test, just return nil
	return nil, nil
}

// CreateMetricsReceiver creates a metrics receiver based on this config.
func (f *MultiProtoReceiverFactory) CreateMetricsReceiver(
	_ context.Context,
	_ component.ReceiverCreateParams,
	_ configmodels.Receiver,
	_ consumer.MetricsConsumer,
) (component.MetricsReceiver, error) {
	// Not used for this test, just return nil
	return nil, nil
}

// CreateMetricsReceiver creates a metrics receiver based on this config.
func (f *MultiProtoReceiverFactory) CreateLogsReceiver(
	_ context.Context,
	_ component.ReceiverCreateParams,
	_ configmodels.Receiver,
	_ consumer.LogsConsumer,
) (component.LogsReceiver, error) {
	// Not used for this test, just return nil
	return nil, nil
}

// ExampleComponents registers example factories. This is only used by tests.
func ExampleComponents() (
	factories component.Factories,
	err error,
) {
	if factories.Extensions, err = component.MakeExtensionFactoryMap(&ExampleExtensionFactory{}); err != nil {
		return
	}

	factories.Receivers, err = component.MakeReceiverFactoryMap(
		&ExampleReceiverFactory{},
		&MultiProtoReceiverFactory{},
	)
	if err != nil {
		return
	}

	factories.Exporters, err = component.MakeExporterFactoryMap(&ExampleExporterFactory{})
	if err != nil {
		return
	}

	factories.Processors, err = component.MakeProcessorFactoryMap(&ExampleProcessorFactory{})

	return
}
