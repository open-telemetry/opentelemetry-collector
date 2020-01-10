// Copyright 2019, OpenTelemetry Authors
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

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configerror"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter"
	"github.com/open-telemetry/opentelemetry-collector/extension"
	"github.com/open-telemetry/opentelemetry-collector/processor"
	"github.com/open-telemetry/opentelemetry-collector/receiver"
)

// ExampleReceiver is for testing purposes. We are defining an example config and factory
// for "examplereceiver" receiver type.
type ExampleReceiver struct {
	configmodels.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	ExtraSetting                  string                   `mapstructure:"extra"`
	ExtraMapSetting               map[string]string        `mapstructure:"extra_map"`
	ExtraListSetting              []string                 `mapstructure:"extra_list"`

	// FailTraceCreation causes CreateTraceReceiver to fail. Useful for testing.
	FailTraceCreation bool `mapstructure:"-"`

	// FailMetricsCreation causes CreateTraceReceiver to fail. Useful for testing.
	FailMetricsCreation bool `mapstructure:"-"`
}

// ExampleReceiverFactory is factory for ExampleReceiver.
type ExampleReceiverFactory struct {
}

// Type gets the type of the Receiver config created by this factory.
func (f *ExampleReceiverFactory) Type() string {
	return "examplereceiver"
}

// CustomUnmarshaler returns nil because we don't need custom unmarshaling for this factory.
func (f *ExampleReceiverFactory) CustomUnmarshaler() receiver.CustomUnmarshaler {
	return nil
}

// CreateDefaultConfig creates the default configuration for the Receiver.
func (f *ExampleReceiverFactory) CreateDefaultConfig() configmodels.Receiver {
	return &ExampleReceiver{
		ReceiverSettings: configmodels.ReceiverSettings{
			TypeVal:  "examplereceiver",
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
	nextConsumer consumer.TraceConsumer,
) (receiver.TraceReceiver, error) {
	if cfg.(*ExampleReceiver).FailTraceCreation {
		return nil, configerror.ErrDataTypeIsNotSupported
	}

	// There must be one receiver for both metrics and traces. We maintain a map of
	// receivers per config.

	// Check to see if there is already a receiver for this config.
	receiver, ok := exampleReceivers[cfg]
	if !ok {
		receiver = &ExampleReceiverProducer{}
		// Remember the receiver in the map
		exampleReceivers[cfg] = receiver
	}
	receiver.TraceConsumer = nextConsumer

	return receiver, nil
}

// CreateMetricsReceiver creates a metrics receiver based on this config.
func (f *ExampleReceiverFactory) CreateMetricsReceiver(
	logger *zap.Logger,
	cfg configmodels.Receiver,
	nextConsumer consumer.MetricsConsumer,
) (receiver.MetricsReceiver, error) {
	if cfg.(*ExampleReceiver).FailMetricsCreation {
		return nil, configerror.ErrDataTypeIsNotSupported
	}

	// There must be one receiver for both metrics and traces. We maintain a map of
	// receivers per config.

	// Check to see if there is already a receiver for this config.
	receiver, ok := exampleReceivers[cfg]
	if !ok {
		receiver = &ExampleReceiverProducer{}
		// Remember the receiver in the map
		exampleReceivers[cfg] = receiver
	}
	receiver.MetricsConsumer = nextConsumer

	return receiver, nil
}

// ExampleReceiverProducer allows producing traces and metrics for testing purposes.
type ExampleReceiverProducer struct {
	TraceConsumer   consumer.TraceConsumer
	Started         bool
	Stopped         bool
	MetricsConsumer consumer.MetricsConsumer
}

// TraceSource returns the name of the trace data source.
func (erp *ExampleReceiverProducer) TraceSource() string {
	return ""
}

// Start tells the receiver to start its processing.
func (erp *ExampleReceiverProducer) Start(host component.Host) error {
	erp.Started = true
	return nil
}

// Shutdown tells the receiver that should stop reception,
func (erp *ExampleReceiverProducer) Shutdown() error {
	erp.Stopped = true
	return nil
}

// MetricsSource returns the name of the metrics data source.
func (erp *ExampleReceiverProducer) MetricsSource() string {
	return ""
}

// This is the map of already created example receivers for particular configurations.
// We maintain this map because the Factory is asked trace and metric receivers separately
// when it gets CreateTraceReceiver() and CreateMetricsReceiver() but they must not
// create separate objects, they must use one Receiver object per configuration.
var exampleReceivers = map[configmodels.Receiver]*ExampleReceiverProducer{}

// MultiProtoReceiver is for testing purposes. We are defining an example multi protocol
// config and factory for "multireceiver" receiver type.
type MultiProtoReceiver struct {
	TypeVal   string                              `mapstructure:"-"`
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
func (rs *MultiProtoReceiver) Type() string {
	return rs.TypeVal
}

// SetType sets the receiver type.
func (rs *MultiProtoReceiver) SetType(typeStr string) {
	rs.TypeVal = typeStr
}

// IsEnabled returns true if the entity is enabled.
func (rs *MultiProtoReceiver) IsEnabled() bool {
	for _, p := range rs.Protocols {
		if !p.Disabled {
			// If any protocol is enabled then the receiver as a whole should be enabled.
			return true
		}
	}
	// All protocols are disabled so the entire receiver can be disabled.
	return false
}

// MultiProtoReceiverOneCfg is multi proto receiver config.
type MultiProtoReceiverOneCfg struct {
	Disabled     bool   `mapstructure:"disabled"`
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
func (f *MultiProtoReceiverFactory) CustomUnmarshaler() receiver.CustomUnmarshaler {
	return nil
}

// CreateDefaultConfig creates the default configuration for the Receiver.
func (f *MultiProtoReceiverFactory) CreateDefaultConfig() configmodels.Receiver {
	return &MultiProtoReceiver{
		TypeVal: "multireceiver",
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
	nextConsumer consumer.TraceConsumer,
) (receiver.TraceReceiver, error) {
	// Not used for this test, just return nil
	return nil, nil
}

// CreateMetricsReceiver creates a metrics receiver based on this config.
func (f *MultiProtoReceiverFactory) CreateMetricsReceiver(
	logger *zap.Logger,
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
	ExtraInt                      int32                    `mapstructure:"extra_int"`
	ExtraSetting                  string                   `mapstructure:"extra"`
	ExtraMapSetting               map[string]string        `mapstructure:"extra_map"`
	ExtraListSetting              []string                 `mapstructure:"extra_list"`
	ExporterShutdown              bool
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
		ExporterSettings: configmodels.ExporterSettings{},
		ExtraSetting:     "some export string",
		ExtraMapSetting:  nil,
		ExtraListSetting: nil,
	}
}

// CreateTraceExporter creates a trace exporter based on this config.
func (f *ExampleExporterFactory) CreateTraceExporter(logger *zap.Logger, cfg configmodels.Exporter) (exporter.TraceExporter, error) {
	return &ExampleExporterConsumer{}, nil
}

// CreateMetricsExporter creates a metrics exporter based on this config.
func (f *ExampleExporterFactory) CreateMetricsExporter(logger *zap.Logger, cfg configmodels.Exporter) (exporter.MetricsExporter, error) {
	return &ExampleExporterConsumer{}, nil
}

// ExampleExporterConsumer stores consumed traces and metrics for testing purposes.
type ExampleExporterConsumer struct {
	Traces           []consumerdata.TraceData
	Metrics          []consumerdata.MetricsData
	ExporterStarted  bool
	ExporterShutdown bool
}

// Start tells the exporter to start. The exporter may prepare for exporting
// by connecting to the endpoint. Host parameter can be used for communicating
// with the host after Start() has already returned.
func (exp *ExampleExporterConsumer) Start(host component.Host) error {
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

// Name returns the name of the exporter.
func (exp *ExampleExporterConsumer) Name() string {
	return "exampleexporter"
}

// Shutdown is invoked during shutdown.
func (exp *ExampleExporterConsumer) Shutdown() error {
	exp.ExporterShutdown = true
	return nil
}

// ExampleProcessor is for testing purposes. We are defining an example config and factory
// for "exampleprocessor" processor type.
type ExampleProcessor struct {
	configmodels.ProcessorSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	ExtraSetting                   string                   `mapstructure:"extra"`
	ExtraMapSetting                map[string]string        `mapstructure:"extra_map"`
	ExtraListSetting               []string                 `mapstructure:"extra_list"`
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
		ProcessorSettings: configmodels.ProcessorSettings{},
		ExtraSetting:      "some export string",
		ExtraMapSetting:   nil,
		ExtraListSetting:  nil,
	}
}

// CreateTraceProcessor creates a trace processor based on this config.
func (f *ExampleProcessorFactory) CreateTraceProcessor(
	logger *zap.Logger,
	nextConsumer consumer.TraceConsumer,
	cfg configmodels.Processor,
) (processor.TraceProcessor, error) {
	return nil, configerror.ErrDataTypeIsNotSupported
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (f *ExampleProcessorFactory) CreateMetricsProcessor(
	logger *zap.Logger,
	nextConsumer consumer.MetricsConsumer,
	cfg configmodels.Processor,
) (processor.MetricsProcessor, error) {
	return nil, configerror.ErrDataTypeIsNotSupported
}

// ExampleExtension is for testing purposes. We are defining an example config and factory
// for "exampleextension" extension type.
type ExampleExtension struct {
	configmodels.ExtensionSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	ExtraSetting                   string                   `mapstructure:"extra"`
	ExtraMapSetting                map[string]string        `mapstructure:"extra_map"`
	ExtraListSetting               []string                 `mapstructure:"extra_list"`
}

// ExampleExtensionFactory is factory for ExampleExtension.
type ExampleExtensionFactory struct {
}

// Type gets the type of the Extension config created by this factory.
func (f *ExampleExtensionFactory) Type() string {
	return "exampleextension"
}

// CreateDefaultConfig creates the default configuration for the Extension.
func (f *ExampleExtensionFactory) CreateDefaultConfig() configmodels.Extension {
	return &ExampleExtension{
		ExtensionSettings: configmodels.ExtensionSettings{},
		ExtraSetting:      "extra string setting",
		ExtraMapSetting:   nil,
		ExtraListSetting:  nil,
	}
}

// CreateExtension creates an Extension based on this config.
func (f *ExampleExtensionFactory) CreateExtension(
	logger *zap.Logger,
	cfg configmodels.Extension,
) (extension.ServiceExtension, error) {
	return nil, fmt.Errorf("cannot create %q extension type", f.Type())
}

var _ (extension.Factory) = (*ExampleExtensionFactory)(nil)

// ExampleComponents registers example factories. This is only used by tests.
func ExampleComponents() (
	factories Factories,
	err error,
) {
	if factories.Extensions, err = extension.Build(&ExampleExtensionFactory{}); err != nil {
		return
	}

	factories.Receivers, err = receiver.Build(
		&ExampleReceiverFactory{},
		&MultiProtoReceiverFactory{},
	)
	if err != nil {
		return
	}

	factories.Exporters, err = exporter.Build(&ExampleExporterFactory{})
	if err != nil {
		return
	}

	factories.Processors, err = processor.Build(&ExampleProcessorFactory{})

	return
}
