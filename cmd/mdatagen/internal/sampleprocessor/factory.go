// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sampleprocessor // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/sampleprocessor"

import (
	"context"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal/sampleprocessor/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
)

// NewFactory returns a receiver.Factory for sample receiver.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		metadata.Type,
		func() component.Config { return &struct{}{} },
		processor.WithTraces(createTracesProcessor, metadata.TracesStability),
		processor.WithMetrics(createMetricsProcessor, metadata.MetricsStability),
		processor.WithLogs(createLogsProcessor, metadata.LogsStability))
}

func createTracesProcessor(context.Context, processor.Settings, component.Config, consumer.Traces) (processor.Traces, error) {
	return nopInstance, nil
}

func createMetricsProcessor(context.Context, processor.Settings, component.Config, consumer.Metrics) (processor.Metrics, error) {
	return nopInstance, nil
}

func createLogsProcessor(context.Context, processor.Settings, component.Config, consumer.Logs) (processor.Logs, error) {
	return nopInstance, nil
}

var nopInstance = &nopProcessor{}

type nopProcessor struct {
	component.StartFunc
	component.ShutdownFunc
}

func (n nopProcessor) ConsumeTraces(context.Context, ptrace.Traces) error {
	return nil
}

func (n nopProcessor) ConsumeLogs(context.Context, plog.Logs) error {
	return nil
}

func (n nopProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (n nopProcessor) ConsumeMetrics(context.Context, pmetric.Metrics) error {
	return nil
}
