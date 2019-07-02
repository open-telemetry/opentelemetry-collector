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

package factories

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/models"
	"github.com/open-telemetry/opentelemetry-service/processor"
)

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
	CreateDefaultConfig() models.Exporter

	// CreateTraceExporter creates a trace exporter based on this config.
	CreateTraceExporter(logger *zap.Logger, cfg models.Exporter) (consumer.TraceConsumer, StopFunc, error)

	// CreateMetricsExporter creates a metrics exporter based on this config.
	CreateMetricsExporter(logger *zap.Logger, cfg models.Exporter) (consumer.MetricsConsumer, StopFunc, error)
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
	CreateDefaultConfig() models.Processor

	// CreateTraceProcessor creates a trace processor based on this config.
	// If the processor type does not support tracing or if the config is not valid
	// error will be returned instead.
	CreateTraceProcessor(logger *zap.Logger, nextConsumer consumer.TraceConsumer,
		cfg models.Processor) (processor.TraceProcessor, error)

	// CreateMetricsProcessor creates a metrics processor based on this config.
	// If the processor type does not support metrics or if the config is not valid
	// error will be returned instead.
	CreateMetricsProcessor(logger *zap.Logger, nextConsumer consumer.MetricsConsumer,
		cfg models.Processor) (processor.MetricsProcessor, error)
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
