// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testcomponents // import "go.opentelemetry.io/collector/service/internal/testcomponents"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/xprocessor"
)

var procType = component.MustNewType("exampleprocessor")

// ExampleProcessorFactory is factory for ExampleProcessor.
var ExampleProcessorFactory = xprocessor.NewFactory(
	procType,
	createDefaultConfig,
	xprocessor.WithTraces(createTracesProcessor, component.StabilityLevelDevelopment),
	xprocessor.WithMetrics(createMetricsProcessor, component.StabilityLevelDevelopment),
	xprocessor.WithLogs(createLogsProcessor, component.StabilityLevelDevelopment),
	xprocessor.WithProfiles(createProfilesProcessor, component.StabilityLevelDevelopment),
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

func createProfilesProcessor(_ context.Context, set processor.Settings, _ component.Config, nextConsumer xconsumer.Profiles) (xprocessor.Profiles, error) {
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
	xconsumer.ConsumeProfilesFunc
	mutatesData bool
}

func (ep *ExampleProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: ep.mutatesData}
}
