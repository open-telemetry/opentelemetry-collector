// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processortest // import "go.opentelemetry.io/collector/processor/processortest"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor"
)

// NewUnhealthyProcessorCreateSettings returns a new nop settings for Create*Processor functions.
func NewUnhealthyProcessorCreateSettings() processor.CreateSettings {
	return processor.CreateSettings{
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

// NewUnhealthyProcessorFactory returns a component.ProcessorFactory that constructs nop processors.
func NewUnhealthyProcessorFactory() processor.Factory {
	return processor.NewFactory(
		"unhealthy",
		func() component.Config {
			return &struct{}{}
		},
		processor.WithTraces(createUnhealthyTracesProcessor, component.StabilityLevelStable),
		processor.WithMetrics(createUnhealthyMetricsProcessor, component.StabilityLevelStable),
		processor.WithLogs(createUnhealthyLogsProcessor, component.StabilityLevelStable),
	)
}

func createUnhealthyTracesProcessor(context.Context, processor.CreateSettings, component.Config, consumer.Traces) (processor.Traces, error) {
	return unhealthyProcessorInstance, nil
}

func createUnhealthyMetricsProcessor(context.Context, processor.CreateSettings, component.Config, consumer.Metrics) (processor.Metrics, error) {
	return unhealthyProcessorInstance, nil
}

func createUnhealthyLogsProcessor(context.Context, processor.CreateSettings, component.Config, consumer.Logs) (processor.Logs, error) {
	return unhealthyProcessorInstance, nil
}

var unhealthyProcessorInstance = &unhealthyProcessor{
	Consumer: consumertest.NewNop(),
}

// unhealthyProcessor stores consumed traces and metrics for testing purposes.
type unhealthyProcessor struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}

func (unhealthyProcessor) Start(_ context.Context, host component.Host) error {
	go func() {
		evt, _ := component.NewStatusEvent(component.StatusError)
		host.ReportComponentStatus(evt)
	}()
	return nil
}
