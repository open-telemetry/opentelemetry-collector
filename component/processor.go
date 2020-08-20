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

package component

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
)

// Processor defines the common functions that must be implemented by TraceProcessor
// and MetricsProcessor.
type Processor interface {
	Component

	// GetCapabilities must return the capabilities of the processor.
	GetCapabilities() ProcessorCapabilities
}

// TraceProcessorBase is a common interface for TraceProcessor and TraceProcessorOld
type TraceProcessorBase interface {
	Processor
}

// TraceProcessorOld is a processor that can consume old-style traces.
type TraceProcessorOld interface {
	consumer.TraceConsumerOld
	TraceProcessorBase
}

// TraceProcessor is a processor that can consume traces.
type TraceProcessor interface {
	consumer.TraceConsumer
	TraceProcessorBase
}

// MetricsProcessorBase is a common interface for MetricsProcessor and MetricsProcessorV2
type MetricsProcessorBase interface {
	Processor
}

// MetricsProcessor is a processor that can consume old-style metrics.
type MetricsProcessorOld interface {
	consumer.MetricsConsumerOld
	MetricsProcessorBase
}

// MetricsProcessor is a processor that can consume metrics.
type MetricsProcessor interface {
	consumer.MetricsConsumer
	MetricsProcessorBase
}

// LogsProcessor is a processor that can consume logs.
type LogsProcessor interface {
	Processor
	consumer.LogsConsumer
}

// ProcessorCapabilities describes the capabilities of a Processor.
type ProcessorCapabilities struct {
	// MutatesConsumedData is set to true if Consume* function of the
	// processor modifies the input TraceData or MetricsData argument.
	// Processors which modify the input data MUST set this flag to true. If the processor
	// does not modify the data it MUST set this flag to false. If the processor creates
	// a copy of the data before modifying then this flag can be safely set to false.
	MutatesConsumedData bool
}

// ProcessorFactoryBase defines the common functions for all processor factories.
type ProcessorFactoryBase interface {
	Factory

	// CreateDefaultConfig creates the default configuration for the Processor.
	// This method can be called multiple times depending on the pipeline
	// configuration and should not cause side-effects that prevent the creation
	// of multiple instances of the Processor.
	// The object returned by this method needs to pass the checks implemented by
	// 'configcheck.ValidateConfig'. It is recommended to have such check in the
	// tests of any implementation of the Factory interface.
	CreateDefaultConfig() configmodels.Processor
}

// ProcessorFactoryOld is factory interface for processors.
type ProcessorFactoryOld interface {
	ProcessorFactoryBase

	// CreateTraceProcessor creates a trace processor based on this config.
	// If the processor type does not support tracing or if the config is not valid
	// error will be returned instead.
	CreateTraceProcessor(logger *zap.Logger, nextConsumer consumer.TraceConsumerOld,
		cfg configmodels.Processor) (TraceProcessorOld, error)

	// CreateMetricsProcessor creates a metrics processor based on this config.
	// If the processor type does not support metrics or if the config is not valid
	// error will be returned instead.
	CreateMetricsProcessor(logger *zap.Logger, nextConsumer consumer.MetricsConsumerOld,
		cfg configmodels.Processor) (MetricsProcessorOld, error)
}

// ProcessorCreateParams is passed to Create* functions in ProcessorFactory.
type ProcessorCreateParams struct {
	// Logger that the factory can use during creation and can pass to the created
	// component to be used later as well.
	Logger *zap.Logger
}

// ProcessorFactory is factory interface for processors. This is the
// new factory type that can create new style processors.
type ProcessorFactory interface {
	ProcessorFactoryBase

	// CreateTraceProcessor creates a trace processor based on this config.
	// If the processor type does not support tracing or if the config is not valid
	// error will be returned instead.
	CreateTraceProcessor(ctx context.Context, params ProcessorCreateParams,
		nextConsumer consumer.TraceConsumer, cfg configmodels.Processor) (TraceProcessor, error)

	// CreateMetricsProcessor creates a metrics processor based on this config.
	// If the processor type does not support metrics or if the config is not valid
	// error will be returned instead.
	CreateMetricsProcessor(ctx context.Context, params ProcessorCreateParams,
		nextConsumer consumer.MetricsConsumer, cfg configmodels.Processor) (MetricsProcessor, error)
}

// LogsProcessorFactory can create LogsProcessor.
type LogsProcessorFactory interface {
	ProcessorFactoryBase

	// CreateLogsProcessor creates a processor based on the config.
	// If the processor type does not support logs or if the config is not valid
	// error will be returned instead.
	CreateLogsProcessor(
		ctx context.Context,
		params ProcessorCreateParams,
		cfg configmodels.Processor,
		nextConsumer consumer.LogsConsumer,
	) (LogsProcessor, error)
}
