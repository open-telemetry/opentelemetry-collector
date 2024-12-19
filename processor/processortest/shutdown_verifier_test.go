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
		processor.WithTraces(createPassthroughTraces, component.StabilityLevelStable),
		processor.WithMetrics(createPassthroughMetrics, component.StabilityLevelStable),
		processor.WithLogs(createPassthroughLogs, component.StabilityLevelStable),
	)
	VerifyShutdown(t, f, &struct{}{})
}

func TestShutdownVerifierLogsOnly(t *testing.T) {
	f := processor.NewFactory(
		component.MustNewType("passthrough"),
		func() component.Config { return struct{}{} },
		processor.WithLogs(createPassthroughLogs, component.StabilityLevelStable),
	)
	VerifyShutdown(t, f, &struct{}{})
}

func TestShutdownVerifierMetricsOnly(t *testing.T) {
	f := processor.NewFactory(
		component.MustNewType("passthrough"),
		func() component.Config { return struct{}{} },
		processor.WithMetrics(createPassthroughMetrics, component.StabilityLevelStable),
	)
	VerifyShutdown(t, f, &struct{}{})
}

func TestShutdownVerifierTracesOnly(t *testing.T) {
	f := processor.NewFactory(
		component.MustNewType("passthrough"),
		func() component.Config { return struct{}{} },
		processor.WithTraces(createPassthroughTraces, component.StabilityLevelStable),
	)
	VerifyShutdown(t, f, &struct{}{})
}

type passthrough struct {
	processor.Traces
	nextLogs    consumer.Logs
	nextMetrics consumer.Metrics
	nextTraces  consumer.Traces
}

func (passthrough) Start(context.Context, component.Host) error {
	return nil
}

func (passthrough) Shutdown(context.Context) error {
	return nil
}

func (passthrough) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}

func createPassthroughLogs(_ context.Context, _ processor.Settings, _ component.Config, logs consumer.Logs) (processor.Logs, error) {
	return passthrough{
		nextLogs: logs,
	}, nil
}

func createPassthroughMetrics(_ context.Context, _ processor.Settings, _ component.Config, metrics consumer.Metrics) (processor.Metrics, error) {
	return passthrough{
		nextMetrics: metrics,
	}, nil
}

func createPassthroughTraces(_ context.Context, _ processor.Settings, _ component.Config, traces consumer.Traces) (processor.Traces, error) {
	return passthrough{
		nextTraces: traces,
	}, nil
}

func (p passthrough) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	return p.nextTraces.ConsumeTraces(ctx, td)
}

func (p passthrough) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	return p.nextMetrics.ConsumeMetrics(ctx, md)
}

func (p passthrough) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	return p.nextLogs.ConsumeLogs(ctx, ld)
}
