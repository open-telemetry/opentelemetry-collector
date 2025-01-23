// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processortest // import "go.opentelemetry.io/collector/processor/processortest"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor"
)

// NewUnhealthyProcessorFactory returns a processor.Factory that constructs nop processors.
func NewUnhealthyProcessorFactory() processor.Factory {
	return processor.NewFactory(
		component.MustNewType("unhealthy"),
		func() component.Config {
			return &struct{}{}
		},
		processor.WithTraces(createUnhealthyTraces, component.StabilityLevelStable),
		processor.WithMetrics(createUnhealthyMetrics, component.StabilityLevelStable),
		processor.WithLogs(createUnhealthyLogs, component.StabilityLevelStable),
	)
}

func createUnhealthyTraces(_ context.Context, set processor.Settings, _ component.Config, _ consumer.Traces) (processor.Traces, error) {
	return &unhealthy{
		Consumer:  consumertest.NewNop(),
		telemetry: set.TelemetrySettings,
	}, nil
}

func createUnhealthyMetrics(_ context.Context, set processor.Settings, _ component.Config, _ consumer.Metrics) (processor.Metrics, error) {
	return &unhealthy{
		Consumer:  consumertest.NewNop(),
		telemetry: set.TelemetrySettings,
	}, nil
}

func createUnhealthyLogs(_ context.Context, set processor.Settings, _ component.Config, _ consumer.Logs) (processor.Logs, error) {
	return &unhealthy{
		Consumer:  consumertest.NewNop(),
		telemetry: set.TelemetrySettings,
	}, nil
}

type unhealthy struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
	telemetry component.TelemetrySettings
}

func (p unhealthy) Start(_ context.Context, host component.Host) error {
	go func() {
		componentstatus.ReportStatus(host, componentstatus.NewEvent(componentstatus.StatusRecoverableError))
	}()
	return nil
}
