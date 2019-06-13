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

package factories

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/viper"

	"github.com/census-instrumentation/opencensus-service/consumer"
	"github.com/census-instrumentation/opencensus-service/internal/configmodels"
	"github.com/census-instrumentation/opencensus-service/processor"
	"github.com/census-instrumentation/opencensus-service/receiver"
)

///////////////////////////////////////////////////////////////////////////////
// Receiver factory and its registry.

// ReceiverFactory is factory interface for receivers.
type ReceiverFactory interface {
	// Type gets the type of the Receiver created by this factory.
	Type() string

	// CreateDefaultConfig creates the default configuration for the Receiver.
	CreateDefaultConfig() configmodels.Receiver

	// CustomUnmarshaler returns a custom unmarshaler for the configuration or nil if
	// there is no need for custom unmarshaling. This is typically used if viper.Unmarshal()
	// is not sufficient to unmarshal correctly.
	CustomUnmarshaler() CustomUnmarshaler

	// CreateTraceReceiver creates a trace receiver based on this config.
	// If the receiver type does not support tracing or if the config is not valid
	// error will be returned instead.
	CreateTraceReceiver(ctx context.Context, cfg configmodels.Receiver,
		nextConsumer consumer.TraceConsumer) (receiver.TraceReceiver, error)

	// CreateMetricsReceiver creates a metrics receiver based on this config.
	// If the receiver type does not support metrics or if the config is not valid
	// error will be returned instead.
	CreateMetricsReceiver(cfg configmodels.Receiver,
		consumer consumer.MetricsConsumer) (receiver.MetricsReceiver, error)
}

// ErrDataTypeIsNotSupported can be returned by CreateTraceReceiver or
// CreateMetricsReceiver if the particular telemetry data type is not supported
// by the receiver.
var ErrDataTypeIsNotSupported = errors.New("telemetry type is not supported")

// CustomUnmarshaler is a function that un-marshals a viper data into a config struct
// in a custom way.
type CustomUnmarshaler func(v *viper.Viper, viperKey string, intoCfg interface{}) error

// List of registered receiver factories.
var receiverFactories = make(map[string]ReceiverFactory)

// RegisterReceiverFactory registers a receiver factory.
func RegisterReceiverFactory(factory ReceiverFactory) error {
	if receiverFactories[factory.Type()] != nil {
		panic(fmt.Sprintf("duplicate receiver factory %q", factory.Type()))
	}

	receiverFactories[factory.Type()] = factory
	return nil
}

// GetReceiverFactory gets a receiver factory by type string.
func GetReceiverFactory(typeStr string) ReceiverFactory {
	return receiverFactories[typeStr]
}

///////////////////////////////////////////////////////////////////////////////
// Exporter factory and its registry.

// StopFunc is a function that can be called to stop an exporter that
// was created previously.
type StopFunc func() error

// ExporterFactory is factory interface for exporters. Note: only configuration-related
// functionality exists for now. We will add more factory functionality in the future.
type ExporterFactory interface {
	// Type gets the type of the Exporter created by this factory.
	Type() string

	// CreateDefaultConfig creates the default configuration for the Exporter.
	CreateDefaultConfig() configmodels.Exporter

	// CreateTraceExporter creates a trace exporter based on this config.
	CreateTraceExporter(cfg configmodels.Exporter) (consumer.TraceConsumer, StopFunc, error)

	// CreateMetricsExporter creates a metrics exporter based on this config.
	CreateMetricsExporter(cfg configmodels.Exporter) (consumer.MetricsConsumer, StopFunc, error)
}

// List of registered exporter factories.
var exporterFactories = make(map[string]ExporterFactory)

// RegisterExporterFactory registers a exporter factory.
func RegisterExporterFactory(factory ExporterFactory) error {
	if exporterFactories[factory.Type()] != nil {
		panic(fmt.Sprintf("duplicate exporter factory %q", factory.Type()))
	}

	exporterFactories[factory.Type()] = factory
	return nil
}

// GetExporterFactory gets a exporter factory by type string.
func GetExporterFactory(typeStr string) ExporterFactory {
	return exporterFactories[typeStr]
}

///////////////////////////////////////////////////////////////////////////////
// Processor factory and its registry.

// ProcessorFactory is factory interface for processors.
type ProcessorFactory interface {
	// Type gets the type of the Processor created by this factory.
	Type() string

	// CreateDefaultConfig creates the default configuration for the Processor.
	CreateDefaultConfig() configmodels.Processor

	// CreateTraceProcessor creates a trace processor based on this config.
	// If the processor type does not support tracing or if the config is not valid
	// error will be returned instead.
	CreateTraceProcessor(nextConsumer consumer.TraceConsumer,
		cfg configmodels.Processor) (processor.TraceProcessor, error)

	// CreateMetricsProcessor creates a metrics processor based on this config.
	// If the processor type does not support metrics or if the config is not valid
	// error will be returned instead.
	CreateMetricsProcessor(nextConsumer consumer.MetricsConsumer,
		cfg configmodels.Processor) (processor.MetricsProcessor, error)
}

// List of registered processor factories.
var processorFactories = make(map[string]ProcessorFactory)

// RegisterProcessorFactory registers a processor factory.
func RegisterProcessorFactory(factory ProcessorFactory) error {
	if processorFactories[factory.Type()] != nil {
		panic(fmt.Sprintf("duplicate processor factory %q", factory.Type()))
	}

	processorFactories[factory.Type()] = factory
	return nil
}

// GetProcessorFactory gets a processor factory by type string.
func GetProcessorFactory(typeStr string) ProcessorFactory {
	return processorFactories[typeStr]
}
