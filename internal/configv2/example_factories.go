// Copyright 2019, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configv2

import (
	"context"

	"github.com/census-instrumentation/opencensus-service/consumer"
	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/internal/configmodels"
	"github.com/census-instrumentation/opencensus-service/internal/factories"
	"github.com/census-instrumentation/opencensus-service/processor"
	"github.com/census-instrumentation/opencensus-service/receiver"
)

// ExampleReceiver is for testing purposes. We are defining an example config and factory
// for "examplereceiver" receiver type.
type ExampleReceiver struct {
	configmodels.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	ExtraSetting                  string                   `mapstructure:"extra"`
}

// ExampleReceiverFactory is factory for ExampleReceiver.
type ExampleReceiverFactory struct {
}

// Type gets the type of the Receiver config created by this factory.
func (f *ExampleReceiverFactory) Type() string {
	return "examplereceiver"
}

// CustomUnmarshaler returns nil because we don't need custom unmarshaling for this factory.
func (f *ExampleReceiverFactory) CustomUnmarshaler() factories.CustomUnmarshaler {
	return nil
}

// CreateDefaultConfig creates the default configuration for the Receiver.
func (f *ExampleReceiverFactory) CreateDefaultConfig() configmodels.Receiver {
	return &ExampleReceiver{
		ReceiverSettings: configmodels.ReceiverSettings{
			Endpoint: "localhost:1000",
			Enabled:  false,
		},
		ExtraSetting: "some string",
	}
}

// CreateTraceReceiver creates a trace receiver based on this config.
func (f *ExampleReceiverFactory) CreateTraceReceiver(
	ctx context.Context,
	cfg configmodels.Receiver,
	nextConsumer consumer.TraceConsumer,
) (receiver.TraceReceiver, error) {
	// Not used for this test, just return nil
	return nil, nil
}

// CreateMetricsReceiver creates a metrics receiver based on this config.
func (f *ExampleReceiverFactory) CreateMetricsReceiver(
	cfg configmodels.Receiver,
	consumer consumer.MetricsConsumer,
) (receiver.MetricsReceiver, error) {
	// Not used for this test, just return nil
	return nil, nil
}

// MultiProtoReceiver is for testing purposes. We are defining an example multi protocol
// config and factory for "multireceiver" receiver type.
type MultiProtoReceiver struct {
	Protocols map[string]MultiProtoReceiverOneCfg `mapstructure:"protocols"`
}

var _ configmodels.Receiver = (*MultiProtoReceiver)(nil)

// MultiProtoReceiverOneCfg is multi proto receiver config.
type MultiProtoReceiverOneCfg struct {
	Enabled      bool   `mapstructure:"enabled"`
	Endpoint     string `mapstructure:"endpoint"`
	ExtraSetting string `mapstructure:"extra"`
}

// MultiProtoReceiverFactory is factory for MultiProtoReceiver.
type MultiProtoReceiverFactory struct {
}

// Type gets the type of the Receiver config created by this factory.
func (f *MultiProtoReceiverFactory) Type() string {
	return "multireceiver"
}

// CustomUnmarshaler returns nil because we don't need custom unmarshaling for this factory.
func (f *MultiProtoReceiverFactory) CustomUnmarshaler() factories.CustomUnmarshaler {
	return nil
}

// CreateDefaultConfig creates the default configuration for the Receiver.
func (f *MultiProtoReceiverFactory) CreateDefaultConfig() configmodels.Receiver {
	return &MultiProtoReceiver{
		Protocols: map[string]MultiProtoReceiverOneCfg{
			"http": {
				Enabled:      false,
				Endpoint:     "example.com:8888",
				ExtraSetting: "extra string 1",
			},
			"tcp": {
				Enabled:      false,
				Endpoint:     "omnition.com:9999",
				ExtraSetting: "extra string 2",
			},
		},
	}
}

// CreateTraceReceiver creates a trace receiver based on this config.
func (f *MultiProtoReceiverFactory) CreateTraceReceiver(
	ctx context.Context,
	cfg configmodels.Receiver,
	nextConsumer consumer.TraceConsumer,
) (receiver.TraceReceiver, error) {
	// Not used for this test, just return nil
	return nil, nil
}

// CreateMetricsReceiver creates a metrics receiver based on this config.
func (f *MultiProtoReceiverFactory) CreateMetricsReceiver(
	cfg configmodels.Receiver,
	consumer consumer.MetricsConsumer,
) (receiver.MetricsReceiver, error) {
	// Not used for this test, just return nil
	return nil, nil
}

// ExampleExporter is for testing purposes. We are defining an example config and factory
// for "exampleexporter" exporter type.
type ExampleExporter struct {
	configmodels.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	ExtraSetting                  string                   `mapstructure:"extra"`
}

// ExampleExporterFactory is factory for ExampleExporter.
type ExampleExporterFactory struct {
}

// Type gets the type of the Exporter config created by this factory.
func (f *ExampleExporterFactory) Type() string {
	return "exampleexporter"
}

// CreateDefaultConfig creates the default configuration for the Exporter.
func (f *ExampleExporterFactory) CreateDefaultConfig() configmodels.Exporter {
	return &ExampleExporter{
		ExporterSettings: configmodels.ExporterSettings{
			Enabled: false,
		},
		ExtraSetting: "some export string",
	}
}

// CreateTraceExporter creates a trace exporter based on this config.
func (f *ExampleExporterFactory) CreateTraceExporter(cfg configmodels.Exporter) (consumer.TraceConsumer, factories.StopFunc, error) {
	return &ExampleExporterConsumer{}, nil, nil
}

// CreateMetricsExporter creates a metrics exporter based on this config.
func (f *ExampleExporterFactory) CreateMetricsExporter(cfg configmodels.Exporter) (consumer.MetricsConsumer, factories.StopFunc, error) {
	return &ExampleExporterConsumer{}, nil, nil
}

// ExampleExporterConsumer stores consumed traces and metrics for testing purposes.
type ExampleExporterConsumer struct {
	Traces  []data.TraceData
	Metrics []data.MetricsData
}

// ConsumeTraceData receives data.TraceData for processing by the TraceConsumer.
func (exp *ExampleExporterConsumer) ConsumeTraceData(ctx context.Context, td data.TraceData) error {
	exp.Traces = append(exp.Traces, td)
	return nil
}

// ConsumeMetricsData receives data.MetricsData for processing by the MetricsConsumer.
func (exp *ExampleExporterConsumer) ConsumeMetricsData(ctx context.Context, md data.MetricsData) error {
	exp.Metrics = append(exp.Metrics, md)
	return nil
}

// ExampleProcessor is for testing purposes. We are defining an example config and factory
// for "exampleprocessor" processor type.
type ExampleProcessor struct {
	configmodels.ProcessorSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	ExtraSetting                   string                   `mapstructure:"extra"`
}

// ExampleProcessorFactory is factory for ExampleProcessor.
type ExampleProcessorFactory struct {
}

// Type gets the type of the Processor config created by this factory.
func (f *ExampleProcessorFactory) Type() string {
	return "exampleprocessor"
}

// CreateDefaultConfig creates the default configuration for the Processor.
func (f *ExampleProcessorFactory) CreateDefaultConfig() configmodels.Processor {
	return &ExampleProcessor{
		ProcessorSettings: configmodels.ProcessorSettings{
			Enabled: false,
		},
		ExtraSetting: "some export string",
	}
}

// CreateTraceProcessor creates a trace processor based on this config.
func (f *ExampleProcessorFactory) CreateTraceProcessor(
	nextConsumer consumer.TraceConsumer,
	cfg configmodels.Processor,
) (processor.TraceProcessor, error) {
	return nil, factories.ErrDataTypeIsNotSupported
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (f *ExampleProcessorFactory) CreateMetricsProcessor(
	nextConsumer consumer.MetricsConsumer,
	cfg configmodels.Processor,
) (processor.MetricsProcessor, error) {
	return nil, factories.ErrDataTypeIsNotSupported
}

// RegisterTestFactories registers example factories. This is only used by tests.
func RegisterTestFactories() error {
	_ = factories.RegisterReceiverFactory(&ExampleReceiverFactory{})
	_ = factories.RegisterReceiverFactory(&MultiProtoReceiverFactory{})
	_ = factories.RegisterExporterFactory(&ExampleExporterFactory{})
	_ = factories.RegisterProcessorFactory(&ExampleProcessorFactory{})
	return nil
}
