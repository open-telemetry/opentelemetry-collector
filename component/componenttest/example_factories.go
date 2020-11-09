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
	"fmt"

	"github.com/spf13/viper"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configerror"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
)

// ExampleReceiver is for testing purposes. We are defining an example config and factory
// for "examplereceiver" receiver type.
type ExampleReceiver struct {
	configmodels.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	// Configures the receiver server protocol.
	confignet.TCPAddr `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct

	ExtraSetting     string            `mapstructure:"extra"`
	ExtraMapSetting  map[string]string `mapstructure:"extra_map"`
	ExtraListSetting []string          `mapstructure:"extra_list"`

	// FailTraceCreation causes CreateTracesReceiver to fail. Useful for testing.
	FailTraceCreation bool `mapstructure:"-"`

	// FailMetricsCreation causes CreateMetricsReceiver to fail. Useful for testing.
	FailMetricsCreation bool `mapstructure:"-"`
}

// ExampleReceiverFactory is factory for ExampleReceiver.
type ExampleReceiverFactory struct {
}

var _ component.ReceiverFactory = (*ExampleReceiverFactory)(nil)

// Type gets the type of the Receiver config created by this factory.
func (f *ExampleReceiverFactory) Type() configmodels.Type {
	return "examplereceiver"
}

// CreateDefaultConfig creates the default configuration for the Receiver.
func (f *ExampleReceiverFactory) CreateDefaultConfig() configmodels.Receiver {
	return &ExampleReceiver{
		ReceiverSettings: configmodels.ReceiverSettings{
			TypeVal: f.Type(),
			NameVal: string(f.Type()),
		},
		TCPAddr: confignet.TCPAddr{
			Endpoint: "localhost:1000",
		},
		ExtraSetting:     "some string",
		ExtraMapSetting:  nil,
		ExtraListSetting: nil,
	}
}

// CustomUnmarshaler implements the deprecated way to provide custom unmarshalers.
func (f *ExampleReceiverFactory) CustomUnmarshaler() component.CustomUnmarshaler {
	return nil
}

// CreateTraceReceiver creates a trace receiver based on this config.
func (f *ExampleReceiverFactory) CreateTracesReceiver(
	_ context.Context,
	_ component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	nextConsumer consumer.TracesConsumer,
) (component.TracesReceiver, error) {
	if cfg.(*ExampleReceiver).FailTraceCreation {
		return nil, configerror.ErrDataTypeIsNotSupported
	}

	receiver := f.createReceiver(cfg)
	receiver.TraceConsumer = nextConsumer

	return receiver, nil
}

func (f *ExampleReceiverFactory) createReceiver(cfg configmodels.Receiver) *ExampleReceiverProducer {
	// There must be one receiver for all data types. We maintain a map of
	// receivers per config.

	// Check to see if there is already a receiver for this config.
	receiver, ok := exampleReceivers[cfg]
	if !ok {
		receiver = &ExampleReceiverProducer{}
		// Remember the receiver in the map
		exampleReceivers[cfg] = receiver
	}

	return receiver
}

// CreateMetricsReceiver creates a metrics receiver based on this config.
func (f *ExampleReceiverFactory) CreateMetricsReceiver(
	_ context.Context,
	_ component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	nextConsumer consumer.MetricsConsumer,
) (component.MetricsReceiver, error) {
	if cfg.(*ExampleReceiver).FailMetricsCreation {
		return nil, configerror.ErrDataTypeIsNotSupported
	}

	receiver := f.createReceiver(cfg)
	receiver.MetricsConsumer = nextConsumer

	return receiver, nil
}

func (f *ExampleReceiverFactory) CreateLogsReceiver(
	_ context.Context,
	_ component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	nextConsumer consumer.LogsConsumer,
) (component.LogsReceiver, error) {
	receiver := f.createReceiver(cfg)
	receiver.LogConsumer = nextConsumer

	return receiver, nil
}

// ExampleReceiverProducer allows producing traces and metrics for testing purposes.
type ExampleReceiverProducer struct {
	Started         bool
	Stopped         bool
	TraceConsumer   consumer.TracesConsumer
	MetricsConsumer consumer.MetricsConsumer
	LogConsumer     consumer.LogsConsumer
}

// Start tells the receiver to start its processing.
func (erp *ExampleReceiverProducer) Start(_ context.Context, _ component.Host) error {
	erp.Started = true
	return nil
}

// Shutdown tells the receiver that should stop reception,
func (erp *ExampleReceiverProducer) Shutdown(context.Context) error {
	erp.Stopped = true
	return nil
}

// This is the map of already created example receivers for particular configurations.
// We maintain this map because the ReceiverFactory is asked trace and metric receivers separately
// when it gets CreateTracesReceiver() and CreateMetricsReceiver() but they must not
// create separate objects, they must use one Receiver object per configuration.
var exampleReceivers = map[configmodels.Receiver]*ExampleReceiverProducer{}

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

// ExampleExporter is for testing purposes. We are defining an example config and factory
// for "exampleexporter" exporter type.
type ExampleExporter struct {
	configmodels.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	ExtraInt                      int32                    `mapstructure:"extra_int"`
	ExtraSetting                  string                   `mapstructure:"extra"`
	ExtraMapSetting               map[string]string        `mapstructure:"extra_map"`
	ExtraListSetting              []string                 `mapstructure:"extra_list"`
}

// ExampleExporterFactory is factory for ExampleExporter.
type ExampleExporterFactory struct {
}

// Type gets the type of the Exporter config created by this factory.
func (f *ExampleExporterFactory) Type() configmodels.Type {
	return "exampleexporter"
}

// CreateDefaultConfig creates the default configuration for the Exporter.
func (f *ExampleExporterFactory) CreateDefaultConfig() configmodels.Exporter {
	return &ExampleExporter{
		ExporterSettings: configmodels.ExporterSettings{
			TypeVal: f.Type(),
			NameVal: string(f.Type()),
		},
		ExtraSetting:     "some export string",
		ExtraMapSetting:  nil,
		ExtraListSetting: nil,
	}
}

// CustomUnmarshaler implements the deprecated way to provide custom unmarshalers.
func (f *ExampleExporterFactory) CustomUnmarshaler() component.CustomUnmarshaler {
	return func(componentViperSection *viper.Viper, intoCfg interface{}) error {
		return componentViperSection.UnmarshalExact(intoCfg)
	}
}

// CreateTraceExporter creates a trace exporter based on this config.
func (f *ExampleExporterFactory) CreateTracesExporter(
	_ context.Context,
	_ component.ExporterCreateParams,
	_ configmodels.Exporter,
) (component.TracesExporter, error) {
	return &ExampleExporterConsumer{}, nil
}

// CreateMetricsExporter creates a metrics exporter based on this config.
func (f *ExampleExporterFactory) CreateMetricsExporter(
	_ context.Context,
	_ component.ExporterCreateParams,
	_ configmodels.Exporter,
) (component.MetricsExporter, error) {
	return &ExampleExporterConsumer{}, nil
}

func (f *ExampleExporterFactory) CreateLogsExporter(
	_ context.Context,
	_ component.ExporterCreateParams,
	_ configmodels.Exporter,
) (component.LogsExporter, error) {
	return &ExampleExporterConsumer{}, nil
}

// ExampleExporterConsumer stores consumed traces and metrics for testing purposes.
type ExampleExporterConsumer struct {
	Traces           []pdata.Traces
	Metrics          []pdata.Metrics
	Logs             []pdata.Logs
	ExporterStarted  bool
	ExporterShutdown bool
}

// Start tells the exporter to start. The exporter may prepare for exporting
// by connecting to the endpoint. Host parameter can be used for communicating
// with the host after Start() has already returned.
func (exp *ExampleExporterConsumer) Start(_ context.Context, _ component.Host) error {
	exp.ExporterStarted = true
	return nil
}

// ConsumeTraceData receives consumerdata.TraceData for processing by the TracesConsumer.
func (exp *ExampleExporterConsumer) ConsumeTraces(_ context.Context, td pdata.Traces) error {
	exp.Traces = append(exp.Traces, td)
	return nil
}

// ConsumeMetricsData receives consumerdata.MetricsData for processing by the MetricsConsumer.
func (exp *ExampleExporterConsumer) ConsumeMetrics(_ context.Context, md pdata.Metrics) error {
	exp.Metrics = append(exp.Metrics, md)
	return nil
}

func (exp *ExampleExporterConsumer) ConsumeLogs(_ context.Context, ld pdata.Logs) error {
	exp.Logs = append(exp.Logs, ld)
	return nil
}

// Shutdown is invoked during shutdown.
func (exp *ExampleExporterConsumer) Shutdown(context.Context) error {
	exp.ExporterShutdown = true
	return nil
}

// ExampleProcessorCfg is for testing purposes. We are defining an example config and factory
// for "exampleprocessor" processor type.
type ExampleProcessorCfg struct {
	configmodels.ProcessorSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	ExtraSetting                   string                   `mapstructure:"extra"`
	ExtraMapSetting                map[string]string        `mapstructure:"extra_map"`
	ExtraListSetting               []string                 `mapstructure:"extra_list"`
}

// ExampleProcessorFactory is factory for ExampleProcessor.
type ExampleProcessorFactory struct {
}

// Type gets the type of the Processor config created by this factory.
func (f *ExampleProcessorFactory) Type() configmodels.Type {
	return "exampleprocessor"
}

// CreateDefaultConfig creates the default configuration for the Processor.
func (f *ExampleProcessorFactory) CreateDefaultConfig() configmodels.Processor {
	return &ExampleProcessorCfg{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: f.Type(),
			NameVal: string(f.Type()),
		},
		ExtraSetting:     "some export string",
		ExtraMapSetting:  nil,
		ExtraListSetting: nil,
	}
}

// CreateTraceProcessor creates a trace processor based on this config.
func (f *ExampleProcessorFactory) CreateTracesProcessor(ctx context.Context, params component.ProcessorCreateParams, cfg configmodels.Processor, nextConsumer consumer.TracesConsumer) (component.TracesProcessor, error) {
	return &ExampleProcessor{nextTraces: nextConsumer}, nil
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (f *ExampleProcessorFactory) CreateMetricsProcessor(ctx context.Context, params component.ProcessorCreateParams, cfg configmodels.Processor, nextConsumer consumer.MetricsConsumer) (component.MetricsProcessor, error) {
	return &ExampleProcessor{nextMetrics: nextConsumer}, nil
}

func (f *ExampleProcessorFactory) CreateLogsProcessor(
	_ context.Context,
	_ component.ProcessorCreateParams,
	_ configmodels.Processor,
	nextConsumer consumer.LogsConsumer,
) (component.LogsProcessor, error) {
	return &ExampleProcessor{nextLogs: nextConsumer}, nil
}

type ExampleProcessor struct {
	nextTraces  consumer.TracesConsumer
	nextMetrics consumer.MetricsConsumer
	nextLogs    consumer.LogsConsumer
}

func (ep *ExampleProcessor) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (ep *ExampleProcessor) Shutdown(_ context.Context) error {
	return nil
}

func (ep *ExampleProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: false}
}

func (ep *ExampleProcessor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	return ep.nextTraces.ConsumeTraces(ctx, td)
}

func (ep *ExampleProcessor) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	return ep.nextMetrics.ConsumeMetrics(ctx, md)
}

func (ep *ExampleProcessor) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
	return ep.nextLogs.ConsumeLogs(ctx, ld)
}

// ExampleExtensionCfg is for testing purposes. We are defining an example config and factory
// for "exampleextension" extension type.
type ExampleExtensionCfg struct {
	configmodels.ExtensionSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	ExtraSetting                   string                   `mapstructure:"extra"`
	ExtraMapSetting                map[string]string        `mapstructure:"extra_map"`
	ExtraListSetting               []string                 `mapstructure:"extra_list"`
}

type ExampleExtension struct {
}

func (e *ExampleExtension) Start(_ context.Context, _ component.Host) error { return nil }

func (e *ExampleExtension) Shutdown(_ context.Context) error { return nil }

// ExampleExtensionFactory is factory for ExampleExtensionCfg.
type ExampleExtensionFactory struct {
	FailCreation bool
}

// Type gets the type of the Extension config created by this factory.
func (f *ExampleExtensionFactory) Type() configmodels.Type {
	return "exampleextension"
}

// CreateDefaultConfig creates the default configuration for the Extension.
func (f *ExampleExtensionFactory) CreateDefaultConfig() configmodels.Extension {
	return &ExampleExtensionCfg{
		ExtensionSettings: configmodels.ExtensionSettings{
			TypeVal: f.Type(),
			NameVal: string(f.Type()),
		},
		ExtraSetting:     "extra string setting",
		ExtraMapSetting:  nil,
		ExtraListSetting: nil,
	}
}

// CreateExtension creates an Extension based on this config.
func (f *ExampleExtensionFactory) CreateExtension(_ context.Context, _ component.ExtensionCreateParams, _ configmodels.Extension) (component.ServiceExtension, error) {
	if f.FailCreation {
		return nil, fmt.Errorf("cannot create %q extension type", f.Type())
	}
	return &ExampleExtension{}, nil
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
