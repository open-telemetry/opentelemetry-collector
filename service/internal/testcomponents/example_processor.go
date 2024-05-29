// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testcomponents // import "go.opentelemetry.io/collector/service/internal/testcomponents"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerlogs"
	"go.opentelemetry.io/collector/consumer/consumermetrics"
	"go.opentelemetry.io/collector/consumer/consumertraces"
	"go.opentelemetry.io/collector/processor"
)

var procType = component.MustNewType("exampleprocessor")

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

func createTracesProcessor(_ context.Context, set processor.CreateSettings, _ component.Config, nextConsumer consumertraces.Traces) (processor.Traces, error) {
	return &ExampleProcessor{
		ConsumeTracesFunc: nextConsumer.ConsumeTraces,
		mutatesData:       set.ID.Name() == "mutate",
	}, nil
}

func createMetricsProcessor(_ context.Context, set processor.CreateSettings, _ component.Config, nextConsumer consumermetrics.Metrics) (processor.Metrics, error) {
	return &ExampleProcessor{
		ConsumeMetricsFunc: nextConsumer.ConsumeMetrics,
		mutatesData:        set.ID.Name() == "mutate",
	}, nil
}

func createLogsProcessor(_ context.Context, set processor.CreateSettings, _ component.Config, nextConsumer consumerlogs.Logs) (processor.Logs, error) {
	return &ExampleProcessor{
		ConsumeLogsFunc: nextConsumer.ConsumeLogs,
		mutatesData:     set.ID.Name() == "mutate",
	}, nil
}

type ExampleProcessor struct {
	componentState
	consumertraces.ConsumeTracesFunc
	consumermetrics.ConsumeMetricsFunc
	consumerlogs.ConsumeLogsFunc
	mutatesData bool
}

func (ep *ExampleProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: ep.mutatesData}
}
