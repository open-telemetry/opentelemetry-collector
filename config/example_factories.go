// Copyright The OpenTelemetry Authors
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

package config

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configerror"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/internal/data"
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

	// FailTraceCreation causes CreateTraceReceiver to fail. Useful for testing.
	FailTraceCreation bool `mapstructure:"-"`

	// FailMetricsCreation causes CreateTraceReceiver to fail. Useful for testing.
	FailMetricsCreation bool `mapstructure:"-"`
}

// ExampleReceiverFactory is factory for ExampleReceiver.
type ExampleReceiverFactory struct {
}

// Type gets the type of the Receiver config created by this factory.
func (f *ExampleReceiverFactory) Type() configmodels.Type {
	return "examplereceiver"
}

// CustomUnmarshaler returns nil because we don't need custom unmarshaling for this factory.
func (f *ExampleReceiverFactory) CustomUnmarshaler() component.CustomUnmarshaler {
	return nil
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

// CreateTraceReceiver creates a trace receiver based on this config.
func (f *ExampleReceiverFactory) CreateTraceReceiver(
	ctx context.Context,
	logger *zap.Logger,
	cfg configmodels.Receiver,
	nextConsumer consumer.TraceConsumerOld,
) (component.TraceReceiver, error) {
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
func (f *ExampleReceiverFactory) CreateMetricsReceiver(ctx context.Context, logger *zap.Logger, cfg configmodels.Receiver, nextConsumer consumer.MetricsConsumerOld) (component.MetricsReceiver, error) {
	if cfg.(*ExampleReceiver).FailMetricsCreation {
		return nil, configerror.ErrDataTypeIsNotSupported
	}

	receiver := f.createReceiver(cfg)
	receiver.MetricsConsumer = nextConsumer

	return receiver, nil
}

func (f *ExampleReceiverFactory) CreateLogReceiver(
	ctx context.Context,
	params component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	nextConsumer consumer.LogConsumer,
) (component.LogReceiver, error) {
	receiver := f.createReceiver(cfg)
	receiver.LogConsumer = nextConsumer

	return receiver, nil
}

// ExampleReceiverProducer allows producing traces and metrics for testing purposes.
type ExampleReceiverProducer struct {
	Started         bool
	Stopped         bool
	TraceConsumer   consumer.TraceConsumerOld
	MetricsConsumer consumer.MetricsConsumerOld
	LogConsumer     consumer.LogConsumer
}

// Start tells the receiver to start its processing.
func (erp *ExampleReceiverProducer) Start(ctx context.Context, host component.Host) error {
	erp.Started = true
	return nil
}

// Shutdown tells the receiver that should stop reception,
func (erp *ExampleReceiverProducer) Shutdown(context.Context) error {
	erp.Stopped = true
	return nil
}

// This is the map of already created example receivers for particular configurations.
// We maintain this map because the ReceiverFactoryBase is asked trace and metric receivers separately
// when it gets CreateTraceReceiver() and CreateMetricsReceiver() but they must not
// create separate objects, they must use one Receiver object per configuration.
var exampleReceivers = map[configmodels.Receiver]*ExampleReceiverProducer{}

// MultiProtoReceiver is for testing purposes. We are defining an example multi protocol
// config and factory for "multireceiver" receiver type.
type MultiProtoReceiver struct {
	TypeVal   configmodels.Type                   `mapstructure:"-"`
	NameVal   string                              `mapstructure:"-"`
	Protocols map[string]MultiProtoReceiverOneCfg `mapstructure:"protocols"`
}

var _ configmodels.Receiver = (*MultiProtoReceiver)(nil)

// Name gets the exporter name.
func (rs *MultiProtoReceiver) Name() string {
	return rs.NameVal
}

// SetName sets the receiver name.
func (rs *MultiProtoReceiver) SetName(name string) {
	rs.NameVal = name
}

// Type sets the receiver type.
func (rs *MultiProtoReceiver) Type() configmodels.Type {
	return rs.TypeVal
}

// SetType sets the receiver type.
func (rs *MultiProtoReceiver) SetType(typeStr configmodels.Type) {
	rs.TypeVal = typeStr
}

// MultiProtoReceiverOneCfg is multi proto receiver config.
type MultiProtoReceiverOneCfg struct {
	Endpoint     string `mapstructure:"endpoint"`
	ExtraSetting string `mapstructure:"extra"`
}

// MultiProtoReceiverFactory is factory for MultiProtoReceiver.
type MultiProtoReceiverFactory struct {
}

// Type gets the type of the Receiver config created by this factory.
func (f *MultiProtoReceiverFactory) Type() configmodels.Type {
	return "multireceiver"
}

// CustomUnmarshaler returns nil because we don't need custom unmarshaling for this factory.
func (f *MultiProtoReceiverFactory) CustomUnmarshaler() component.CustomUnmarshaler {
	return nil
}

// CreateDefaultConfig creates the default configuration for the Receiver.
func (f *MultiProtoReceiverFactory) CreateDefaultConfig() configmodels.Receiver {
	return &MultiProtoReceiver{
		TypeVal: f.Type(),
		NameVal: string(f.Type()),
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
func (f *MultiProtoReceiverFactory) CreateTraceReceiver(
	ctx context.Context,
	logger *zap.Logger,
	cfg configmodels.Receiver,
	nextConsumer consumer.TraceConsumerOld,
) (component.TraceReceiver, error) {
	// Not used for this test, just return nil
	return nil, nil
}

// CreateMetricsReceiver creates a metrics receiver based on this config.
func (f *MultiProtoReceiverFactory) CreateMetricsReceiver(ctx context.Context, logger *zap.Logger, cfg configmodels.Receiver, nextConsumer consumer.MetricsConsumerOld) (component.MetricsReceiver, error) {
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

// CreateTraceExporter creates a trace exporter based on this config.
func (f *ExampleExporterFactory) CreateTraceExporter(logger *zap.Logger, cfg configmodels.Exporter) (component.TraceExporterOld, error) {
	return &ExampleExporterConsumer{}, nil
}

// CreateMetricsExporter creates a metrics exporter based on this config.
func (f *ExampleExporterFactory) CreateMetricsExporter(logger *zap.Logger, cfg configmodels.Exporter) (component.MetricsExporterOld, error) {
	return &ExampleExporterConsumer{}, nil
}

func (f *ExampleExporterFactory) CreateLogExporter(
	ctx context.Context,
	params component.ExporterCreateParams,
	cfg configmodels.Exporter,
) (component.LogExporter, error) {
	return &ExampleExporterConsumer{}, nil
}

// ExampleExporterConsumer stores consumed traces and metrics for testing purposes.
type ExampleExporterConsumer struct {
	Traces           []consumerdata.TraceData
	Metrics          []consumerdata.MetricsData
	Logs             []data.Logs
	ExporterStarted  bool
	ExporterShutdown bool
}

// Start tells the exporter to start. The exporter may prepare for exporting
// by connecting to the endpoint. Host parameter can be used for communicating
// with the host after Start() has already returned.
func (exp *ExampleExporterConsumer) Start(ctx context.Context, host component.Host) error {
	exp.ExporterStarted = true
	return nil
}

// ConsumeTraceData receives consumerdata.TraceData for processing by the TraceConsumer.
func (exp *ExampleExporterConsumer) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	exp.Traces = append(exp.Traces, td)
	return nil
}

// ConsumeMetricsData receives consumerdata.MetricsData for processing by the MetricsConsumer.
func (exp *ExampleExporterConsumer) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	exp.Metrics = append(exp.Metrics, md)
	return nil
}

func (exp *ExampleExporterConsumer) ConsumeLogs(ctx context.Context, ld data.Logs) error {
	exp.Logs = append(exp.Logs, ld)
	return nil
}

// Name returns the name of the exporter.
func (exp *ExampleExporterConsumer) Name() string {
	return "exampleexporter"
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
func (f *ExampleProcessorFactory) CreateTraceProcessor(
	logger *zap.Logger,
	nextConsumer consumer.TraceConsumerOld,
	cfg configmodels.Processor,
) (component.TraceProcessorOld, error) {
	return nil, configerror.ErrDataTypeIsNotSupported
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (f *ExampleProcessorFactory) CreateMetricsProcessor(
	logger *zap.Logger,
	nextConsumer consumer.MetricsConsumerOld,
	cfg configmodels.Processor,
) (component.MetricsProcessorOld, error) {
	return nil, configerror.ErrDataTypeIsNotSupported
}

func (f *ExampleProcessorFactory) CreateLogProcessor(
	ctx context.Context,
	params component.ProcessorCreateParams,
	cfg configmodels.Processor,
	nextConsumer consumer.LogConsumer,
) (component.LogProcessor, error) {
	return &ExampleProcessor{nextConsumer}, nil
}

type ExampleProcessor struct {
	nextConsumer consumer.LogConsumer
}

func (ep *ExampleProcessor) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (ep *ExampleProcessor) Shutdown(ctx context.Context) error {
	return nil
}

func (ep *ExampleProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: false}
}

func (ep *ExampleProcessor) ConsumeLogs(ctx context.Context, ld data.Logs) error {
	return ep.nextConsumer.ConsumeLogs(ctx, ld)
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

func (e *ExampleExtension) Start(ctx context.Context, host component.Host) error { return nil }

func (e *ExampleExtension) Shutdown(ctx context.Context) error { return nil }

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
	factories Factories,
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
