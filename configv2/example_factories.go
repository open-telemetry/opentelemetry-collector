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

package configv2

import (
	"context"

	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/data"
	"github.com/open-telemetry/opentelemetry-service/factories"
	"github.com/open-telemetry/opentelemetry-service/models"
	"github.com/open-telemetry/opentelemetry-service/processor"
	"github.com/open-telemetry/opentelemetry-service/receiver"
	"go.uber.org/zap"
)

// ExampleReceiver is for testing purposes. We are defining an example config and factory
// for "examplereceiver" receiver type.
type ExampleReceiver struct {
	models.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	ExtraSetting            string                   `mapstructure:"extra"`

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
func (f *ExampleReceiverFactory) CreateDefaultConfig() models.Receiver {
	return &ExampleReceiver{
		ReceiverSettings: models.ReceiverSettings{
			TypeVal:  "examplereceiver",
			Endpoint: "localhost:1000",
			Enabled:  false,
		},
		ExtraSetting: "some string",
	}
}

// CreateTraceReceiver creates a trace receiver based on this config.
func (f *ExampleReceiverFactory) CreateTraceReceiver(
	ctx context.Context,
	logger *zap.Logger,
	cfg models.Receiver,
	nextConsumer consumer.TraceConsumer,
) (receiver.TraceReceiver, error) {
	if cfg.(*ExampleReceiver).FailTraceCreation {
		return nil, models.ErrDataTypeIsNotSupported
	}
	return &ExampleReceiverProducer{TraceConsumer: nextConsumer}, nil
}

// CreateMetricsReceiver creates a metrics receiver based on this config.
func (f *ExampleReceiverFactory) CreateMetricsReceiver(
	logger *zap.Logger,
	cfg models.Receiver,
	nextConsumer consumer.MetricsConsumer,
) (receiver.MetricsReceiver, error) {
	if cfg.(*ExampleReceiver).FailMetricsCreation {
		return nil, models.ErrDataTypeIsNotSupported
	}
	return &ExampleReceiverProducer{MetricsConsumer: nextConsumer}, nil
}

// ExampleReceiverProducer allows producing traces and metrics for testing purposes.
type ExampleReceiverProducer struct {
	TraceConsumer   consumer.TraceConsumer
	TraceStarted    bool
	TraceStopped    bool
	MetricsConsumer consumer.MetricsConsumer
	MetricsStarted  bool
	MetricsStopped  bool
}

// TraceSource returns the name of the trace data source.
func (erp *ExampleReceiverProducer) TraceSource() string {
	return ""
}

// StartTraceReception tells the receiver to start its processing.
func (erp *ExampleReceiverProducer) StartTraceReception(host receiver.Host) error {
	erp.TraceStarted = true
	return nil
}

// StopTraceReception tells the receiver that should stop reception,
func (erp *ExampleReceiverProducer) StopTraceReception() error {
	erp.TraceStopped = true
	return nil
}

// MetricsSource returns the name of the metrics data source.
func (erp *ExampleReceiverProducer) MetricsSource() string {
	return ""
}

// StartMetricsReception tells the receiver to start its processing.
func (erp *ExampleReceiverProducer) StartMetricsReception(host receiver.Host) error {
	erp.MetricsStarted = true
	return nil
}

// StopMetricsReception tells the receiver that should stop reception,
func (erp *ExampleReceiverProducer) StopMetricsReception() error {
	erp.MetricsStopped = true
	return nil
}

// MultiProtoReceiver is for testing purposes. We are defining an example multi protocol
// config and factory for "multireceiver" receiver type.
type MultiProtoReceiver struct {
	TypeVal   string                              `mapstructure:"-"`
	NameVal   string                              `mapstructure:"-"`
	Protocols map[string]MultiProtoReceiverOneCfg `mapstructure:"protocols"`
}

var _ models.Receiver = (*MultiProtoReceiver)(nil)

// Name gets the exporter name.
func (rs *MultiProtoReceiver) Name() string {
	return rs.NameVal
}

// SetName sets the exporter name.
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
func (f *MultiProtoReceiverFactory) CustomUnmarshaler() receiver.CustomUnmarshaler {
	return nil
}

// CreateDefaultConfig creates the default configuration for the Receiver.
func (f *MultiProtoReceiverFactory) CreateDefaultConfig() models.Receiver {
	return &MultiProtoReceiver{
		TypeVal: "multireceiver",
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
	logger *zap.Logger,
	cfg models.Receiver,
	nextConsumer consumer.TraceConsumer,
) (receiver.TraceReceiver, error) {
	// Not used for this test, just return nil
	return nil, nil
}

// CreateMetricsReceiver creates a metrics receiver based on this config.
func (f *MultiProtoReceiverFactory) CreateMetricsReceiver(
	logger *zap.Logger,
	cfg models.Receiver,
	consumer consumer.MetricsConsumer,
) (receiver.MetricsReceiver, error) {
	// Not used for this test, just return nil
	return nil, nil
}

// ExampleExporter is for testing purposes. We are defining an example config and factory
// for "exampleexporter" exporter type.
type ExampleExporter struct {
	models.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	ExtraSetting            string                   `mapstructure:"extra"`
}

// ExampleExporterFactory is factory for ExampleExporter.
type ExampleExporterFactory struct {
}

// Type gets the type of the Exporter config created by this factory.
func (f *ExampleExporterFactory) Type() string {
	return "exampleexporter"
}

// CreateDefaultConfig creates the default configuration for the Exporter.
func (f *ExampleExporterFactory) CreateDefaultConfig() models.Exporter {
	return &ExampleExporter{
		ExporterSettings: models.ExporterSettings{
			Enabled: false,
		},
		ExtraSetting: "some export string",
	}
}

// CreateTraceExporter creates a trace exporter based on this config.
func (f *ExampleExporterFactory) CreateTraceExporter(logger *zap.Logger, cfg models.Exporter) (consumer.TraceConsumer, factories.StopFunc, error) {
	return &ExampleExporterConsumer{}, nil, nil
}

// CreateMetricsExporter creates a metrics exporter based on this config.
func (f *ExampleExporterFactory) CreateMetricsExporter(logger *zap.Logger, cfg models.Exporter) (consumer.MetricsConsumer, factories.StopFunc, error) {
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
	models.ProcessorSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	ExtraSetting             string                   `mapstructure:"extra"`
}

// ExampleProcessorFactory is factory for ExampleProcessor.
type ExampleProcessorFactory struct {
}

// Type gets the type of the Processor config created by this factory.
func (f *ExampleProcessorFactory) Type() string {
	return "exampleprocessor"
}

// CreateDefaultConfig creates the default configuration for the Processor.
func (f *ExampleProcessorFactory) CreateDefaultConfig() models.Processor {
	return &ExampleProcessor{
		ProcessorSettings: models.ProcessorSettings{
			Enabled: false,
		},
		ExtraSetting: "some export string",
	}
}

// CreateTraceProcessor creates a trace processor based on this config.
func (f *ExampleProcessorFactory) CreateTraceProcessor(
	logger *zap.Logger,
	nextConsumer consumer.TraceConsumer,
	cfg models.Processor,
) (processor.TraceProcessor, error) {
	return nil, models.ErrDataTypeIsNotSupported
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (f *ExampleProcessorFactory) CreateMetricsProcessor(
	logger *zap.Logger,
	nextConsumer consumer.MetricsConsumer,
	cfg models.Processor,
) (processor.MetricsProcessor, error) {
	return nil, models.ErrDataTypeIsNotSupported
}

// RegisterTestFactories registers example factories. This is only used by tests.
func RegisterTestFactories() error {
	_ = receiver.RegisterReceiverFactory(&ExampleReceiverFactory{})
	_ = receiver.RegisterReceiverFactory(&MultiProtoReceiverFactory{})
	_ = factories.RegisterExporterFactory(&ExampleExporterFactory{})
	_ = factories.RegisterProcessorFactory(&ExampleProcessorFactory{})
	return nil
}
