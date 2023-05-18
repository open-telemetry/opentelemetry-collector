// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testcomponents // import "go.opentelemetry.io/collector/service/internal/testcomponents"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

const procType = "exampleprocessor"

// ExampleProcessorFactory is factory for ExampleProcessor.
var ExampleProcessorFactory = processor.NewFactory(
	procType,
	createDefaultConfig,
	processor.WithTraces(createTracesProcessor, component.StabilityLevelDevelopment),
	processor.WithMetrics(createMetricsProcessor, component.StabilityLevelDevelopment),
	processor.WithLogs(createLogsProcessor, component.StabilityLevelDevelopment))

// CreateDefaultConfig creates the default configuration for the Processor.
func createDefaultConfig() component.Config {
	return &struct{}{}
}

func createTracesProcessor(_ context.Context, set processor.CreateSettings, _ component.Config, nextConsumer consumer.Traces) (processor.Traces, error) {
	return &ExampleProcessor{
		ConsumeTracesFunc: nextConsumer.ConsumeTraces,
		mutatesData:       set.ID.Name() == "mutate",
	}, nil
}

func createMetricsProcessor(_ context.Context, set processor.CreateSettings, _ component.Config, nextConsumer consumer.Metrics) (processor.Metrics, error) {
	return &ExampleProcessor{
		ConsumeMetricsFunc: nextConsumer.ConsumeMetrics,
		mutatesData:        set.ID.Name() == "mutate",
	}, nil
}

func createLogsProcessor(_ context.Context, set processor.CreateSettings, _ component.Config, nextConsumer consumer.Logs) (processor.Logs, error) {
	return &ExampleProcessor{
		ConsumeLogsFunc: nextConsumer.ConsumeLogs,
		mutatesData:     set.ID.Name() == "mutate",
	}, nil
}

type ExampleProcessor struct {
	componentState
	consumer.ConsumeTracesFunc
	consumer.ConsumeMetricsFunc
	consumer.ConsumeLogsFunc
	mutatesData bool
}

func (ep *ExampleProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: ep.mutatesData}
}
