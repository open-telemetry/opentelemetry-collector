// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processortest

import (
	"context"
	"testing"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
)

func TestShutdownVerifier(t *testing.T) {
	f := processor.NewFactory(
		component.MustNewType("passthrough"),
		func() component.Config { return struct{}{} },
		processor.WithTraces(createPassthroughTracesProcessor, component.StabilityLevelStable),
		processor.WithMetrics(createPassthroughMetricsProcessor, component.StabilityLevelStable),
		processor.WithLogs(createPassthroughLogsProcessor, component.StabilityLevelStable),
	)
	VerifyShutdown(t, f, &struct{}{})
}

func TestShutdownVerifierLogsOnly(t *testing.T) {
	f := processor.NewFactory(
		component.MustNewType("passthrough"),
		func() component.Config { return struct{}{} },
		processor.WithLogs(createPassthroughLogsProcessor, component.StabilityLevelStable),
	)
	VerifyShutdown(t, f, &struct{}{})
}

func TestShutdownVerifierMetricsOnly(t *testing.T) {
	f := processor.NewFactory(
		component.MustNewType("passthrough"),
		func() component.Config { return struct{}{} },
		processor.WithMetrics(createPassthroughMetricsProcessor, component.StabilityLevelStable),
	)
	VerifyShutdown(t, f, &struct{}{})
}

func TestShutdownVerifierTracesOnly(t *testing.T) {
	f := processor.NewFactory(
		component.MustNewType("passthrough"),
		func() component.Config { return struct{}{} },
		processor.WithTraces(createPassthroughTracesProcessor, component.StabilityLevelStable),
	)
	VerifyShutdown(t, f, &struct{}{})
}

type passthroughProcessor struct {
	processor.Traces
	nextLogs    consumer.Logs
	nextMetrics consumer.Metrics
	nextTraces  consumer.Traces
}

func (passthroughProcessor) Start(context.Context, component.Host) error {
	return nil
}

func (passthroughProcessor) Shutdown(context.Context) error {
	return nil
}

func (passthroughProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}

func createPassthroughLogsProcessor(_ context.Context, _ processor.CreateSettings, _ component.Config, logs consumer.Logs) (processor.Logs, error) {
	return passthroughProcessor{
		nextLogs: logs,
	}, nil
}

func createPassthroughMetricsProcessor(_ context.Context, _ processor.CreateSettings, _ component.Config, metrics consumer.Metrics) (processor.Metrics, error) {
	return passthroughProcessor{
		nextMetrics: metrics,
	}, nil
}

func createPassthroughTracesProcessor(_ context.Context, _ processor.CreateSettings, _ component.Config, traces consumer.Traces) (processor.Traces, error) {
	return passthroughProcessor{
		nextTraces: traces,
	}, nil
}

func (p passthroughProcessor) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	return p.nextTraces.ConsumeTraces(ctx, td)
}

func (p passthroughProcessor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	return p.nextMetrics.ConsumeMetrics(ctx, md)
}

func (p passthroughProcessor) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	return p.nextLogs.ConsumeLogs(ctx, ld)
}
