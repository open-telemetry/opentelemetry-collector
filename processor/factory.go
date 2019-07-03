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

package processor

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/models"
)

// Factory is factory interface for processors.
type Factory interface {
	// Type gets the type of the Processor created by this factory.
	Type() string

	// CreateDefaultConfig creates the default configuration for the Processor.
	CreateDefaultConfig() models.Processor

	// CreateTraceProcessor creates a trace processor based on this config.
	// If the processor type does not support tracing or if the config is not valid
	// error will be returned instead.
	CreateTraceProcessor(logger *zap.Logger, nextConsumer consumer.TraceConsumer,
		cfg models.Processor) (TraceProcessor, error)

	// CreateMetricsProcessor creates a metrics processor based on this config.
	// If the processor type does not support metrics or if the config is not valid
	// error will be returned instead.
	CreateMetricsProcessor(logger *zap.Logger, nextConsumer consumer.MetricsConsumer,
		cfg models.Processor) (MetricsProcessor, error)
}

// List of registered processor factories.
var processorFactories = make(map[string]Factory)

// RegisterProcessorFactory registers a processor factory.
func RegisterProcessorFactory(factory Factory) error {
	if processorFactories[factory.Type()] != nil {
		panic(fmt.Sprintf("duplicate processor factory %q", factory.Type()))
	}

	processorFactories[factory.Type()] = factory
	return nil
}

// GetProcessorFactory gets a processor factory by type string.
func GetProcessorFactory(typeStr string) Factory {
	return processorFactories[typeStr]
}
