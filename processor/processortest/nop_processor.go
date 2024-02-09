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

var nopType = component.MustNewType("nop")

// NewNopCreateSettings returns a new nop settings for Create*Processor functions.
func NewNopCreateSettings() processor.CreateSettings {
	return processor.CreateSettings{
		ID:                component.NewID(nopType),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

// NewNopFactory returns a component.ProcessorFactory that constructs nop processors.
func NewNopFactory() processor.Factory {
	return processor.NewFactory(
		nopType,
		func() component.Config { return &nopConfig{} },
		processor.WithTraces(createTracesProcessor, component.StabilityLevelStable),
		processor.WithMetrics(createMetricsProcessor, component.StabilityLevelStable),
		processor.WithLogs(createLogsProcessor, component.StabilityLevelStable),
	)
}

func createTracesProcessor(context.Context, processor.CreateSettings, component.Config, consumer.Traces) (processor.Traces, error) {
	return nopInstance, nil
}

func createMetricsProcessor(context.Context, processor.CreateSettings, component.Config, consumer.Metrics) (processor.Metrics, error) {
	return nopInstance, nil
}

func createLogsProcessor(context.Context, processor.CreateSettings, component.Config, consumer.Logs) (processor.Logs, error) {
	return nopInstance, nil
}

type nopConfig struct{}

var nopInstance = &nopProcessor{
	Consumer: consumertest.NewNop(),
}

// nopProcessor acts as a processor for testing purposes.
type nopProcessor struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}

// NewNopBuilder returns a processor.Builder that constructs nop processors.
func NewNopBuilder() *processor.Builder {
	nopFactory := NewNopFactory()
	return processor.NewBuilder(
		map[component.ID]component.Config{component.NewID(nopType): nopFactory.CreateDefaultConfig()},
		map[component.Type]processor.Factory{nopType: nopFactory})
}
