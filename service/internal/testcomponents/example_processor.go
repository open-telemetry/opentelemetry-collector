// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testcomponents // import "go.opentelemetry.io/collector/service/internal/testcomponents"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorprofiles"
)

var procType = component.MustNewType("exampleprocessor")

// ExampleProcessorFactory is factory for ExampleProcessor.
var ExampleProcessorFactory = processorprofiles.NewFactory(
	procType,
	createDefaultConfig,
	processorprofiles.WithTraces(createTracesProcessor, component.StabilityLevelDevelopment),
	processorprofiles.WithMetrics(createMetricsProcessor, component.StabilityLevelDevelopment),
	processorprofiles.WithLogs(createLogsProcessor, component.StabilityLevelDevelopment),
	processorprofiles.WithProfiles(createProfilesProcessor, component.StabilityLevelDevelopment),
)

// CreateDefaultConfig creates the default configuration for the Processor.
func createDefaultConfig() component.Config {
	return &struct{}{}
}

func createTracesProcessor(_ context.Context, set processor.Settings, _ component.Config, nextConsumer consumer.Traces) (processor.Traces, error) {
	return &ExampleProcessor{
		ConsumeTracesFunc: nextConsumer.ConsumeTraces,
		mutatesData:       set.ID.Name() == "mutate",
	}, nil
}

func createMetricsProcessor(_ context.Context, set processor.Settings, _ component.Config, nextConsumer consumer.Metrics) (processor.Metrics, error) {
	return &ExampleProcessor{
		ConsumeMetricsFunc: nextConsumer.ConsumeMetrics,
		mutatesData:        set.ID.Name() == "mutate",
	}, nil
}

func createLogsProcessor(_ context.Context, set processor.Settings, _ component.Config, nextConsumer consumer.Logs) (processor.Logs, error) {
	return &ExampleProcessor{
		ConsumeLogsFunc: nextConsumer.ConsumeLogs,
		mutatesData:     set.ID.Name() == "mutate",
	}, nil
}

func createProfilesProcessor(_ context.Context, set processor.Settings, _ component.Config, nextConsumer consumerprofiles.Profiles) (processorprofiles.Profiles, error) {
	return &ExampleProcessor{
		ConsumeProfilesFunc: nextConsumer.ConsumeProfiles,
		mutatesData:         set.ID.Name() == "mutate",
	}, nil
}

type ExampleProcessor struct {
	componentState
	consumer.ConsumeTracesFunc
	consumer.ConsumeMetricsFunc
	consumer.ConsumeLogsFunc
	consumerprofiles.ConsumeProfilesFunc
	mutatesData bool
}

func (ep *ExampleProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: ep.mutatesData}
}
