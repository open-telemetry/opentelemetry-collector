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

package batchprocessor

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

const (
	// The value of "type" key in configuration.
	typeStr = "batch"

	defaultSendBatchSize = uint32(8192)
	defaultTimeout       = 200 * time.Millisecond
)

// NewFactory returns a new factory for the Batch processor.
func NewFactory() component.ProcessorFactory {
	return processorhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		processorhelper.WithTraces(createTracesProcessor),
		processorhelper.WithMetrics(createMetricsProcessor),
		processorhelper.WithLogs(createLogsProcessor))
}

func createDefaultConfig() config.Processor {
	return &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewID(typeStr)),
		SendBatchSize:     defaultSendBatchSize,
		Timeout:           defaultTimeout,
	}
}

func createTracesProcessor(
	_ context.Context,
	componentSettings component.ComponentSettings,
	cfg config.Processor,
	nextConsumer consumer.Traces,
) (component.TracesProcessor, error) {
	level := configtelemetry.GetMetricsLevelFlagValue()
	return newBatchTracesProcessor(componentSettings, nextConsumer, cfg.(*Config), level)
}

func createMetricsProcessor(
	_ context.Context,
	componentSettings component.ComponentSettings,
	cfg config.Processor,
	nextConsumer consumer.Metrics,
) (component.MetricsProcessor, error) {
	level := configtelemetry.GetMetricsLevelFlagValue()
	return newBatchMetricsProcessor(componentSettings, nextConsumer, cfg.(*Config), level)
}

func createLogsProcessor(
	_ context.Context,
	componentSettings component.ComponentSettings,
	cfg config.Processor,
	nextConsumer consumer.Logs,
) (component.LogsProcessor, error) {
	level := configtelemetry.GetMetricsLevelFlagValue()
	return newBatchLogsProcessor(componentSettings, nextConsumer, cfg.(*Config), level)
}
