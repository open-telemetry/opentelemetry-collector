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

package memorylimiter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

const (
	// The value of "type" Attribute Key in configuration.
	typeStr = "memory_limiter"
)

var processorCapabilities = consumer.Capabilities{MutatesData: false}

// NewFactory returns a new factory for the Memory Limiter processor.
func NewFactory() component.ProcessorFactory {
	return processorhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		processorhelper.WithTraces(createTracesProcessor),
		processorhelper.WithMetrics(createMetricsProcessor),
		processorhelper.WithLogs(createLogsProcessor))
}

// CreateDefaultConfig creates the default configuration for processor. Notice
// that the default configuration is expected to fail for this processor.
func createDefaultConfig() config.Processor {
	return &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewID(typeStr)),
	}
}

func createTracesProcessor(
	_ context.Context,
	set component.ProcessorCreateSettings,
	cfg config.Processor,
	nextConsumer consumer.Traces,
) (component.TracesProcessor, error) {
	ml, err := newMemoryLimiter(set.Logger, cfg.(*Config))
	if err != nil {
		return nil, err
	}
	return processorhelper.NewTracesProcessor(
		cfg,
		nextConsumer,
		ml.processTraces,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithStart(ml.start),
		processorhelper.WithShutdown(ml.shutdown))
}

func createMetricsProcessor(
	_ context.Context,
	set component.ProcessorCreateSettings,
	cfg config.Processor,
	nextConsumer consumer.Metrics,
) (component.MetricsProcessor, error) {
	ml, err := newMemoryLimiter(set.Logger, cfg.(*Config))
	if err != nil {
		return nil, err
	}
	return processorhelper.NewMetricsProcessor(
		cfg,
		nextConsumer,
		ml.processMetrics,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithShutdown(ml.shutdown))
}

func createLogsProcessor(
	_ context.Context,
	set component.ProcessorCreateSettings,
	cfg config.Processor,
	nextConsumer consumer.Logs,
) (component.LogsProcessor, error) {
	ml, err := newMemoryLimiter(set.Logger, cfg.(*Config))
	if err != nil {
		return nil, err
	}
	return processorhelper.NewLogsProcessor(
		cfg,
		nextConsumer,
		ml.processLogs,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithShutdown(ml.shutdown))
}
