// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exportertest

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/receiver"
)

// retryConfig is a configuration to quickly retry failed exports.
var retryConfig = func() configretry.BackOffConfig {
	c := configretry.NewDefaultBackOffConfig()
	c.InitialInterval = time.Millisecond
	return c
}()

// mockReceiver is a receiver with pass-through consumers.
type mockReceiver struct {
	component.StartFunc
	component.ShutdownFunc
	consumer.Traces
	consumer.Metrics
	consumer.Logs
}

// mockFactory is a factory to create exporters sending data to the mockReceiver.
type mockFactory struct {
	mr *mockReceiver
	component.StartFunc
	component.ShutdownFunc
}

func (mef *mockFactory) createMockTraces(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Traces, error) {
	return exporterhelper.NewTraces(ctx, set, cfg,
		mef.mr.ConsumeTraces,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithRetry(retryConfig),
	)
}

func (mef *mockFactory) createMockMetrics(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Metrics, error) {
	return exporterhelper.NewMetrics(ctx, set, cfg,
		mef.mr.ConsumeMetrics,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithRetry(retryConfig),
	)
}

func (mef *mockFactory) createMockLogs(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Logs, error) {
	return exporterhelper.NewLogs(ctx, set, cfg,
		mef.mr.ConsumeLogs,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithRetry(retryConfig),
	)
}

func newMockFactory(mr *mockReceiver) exporter.Factory {
	mef := &mockFactory{mr: mr}
	return exporter.NewFactory(
		component.MustNewType("pass_through_exporter"),
		func() component.Config { return &nopConfig{} },
		exporter.WithTraces(mef.createMockTraces, component.StabilityLevelBeta),
		exporter.WithMetrics(mef.createMockMetrics, component.StabilityLevelBeta),
		exporter.WithLogs(mef.createMockLogs, component.StabilityLevelBeta),
	)
}

func newMockReceiverFactory(mr *mockReceiver) receiver.Factory {
	return receiver.NewFactory(component.MustNewType("pass_through_receiver"),
		func() component.Config { return &nopConfig{} },
		receiver.WithTraces(func(_ context.Context, _ receiver.Settings, _ component.Config, c consumer.Traces) (receiver.Traces, error) {
			mr.Traces = c
			return mr, nil
		}, component.StabilityLevelStable),
		receiver.WithMetrics(func(_ context.Context, _ receiver.Settings, _ component.Config, c consumer.Metrics) (receiver.Metrics, error) {
			mr.Metrics = c
			return mr, nil
		}, component.StabilityLevelStable),
		receiver.WithLogs(func(_ context.Context, _ receiver.Settings, _ component.Config, c consumer.Logs) (receiver.Logs, error) {
			mr.Logs = c
			return mr, nil
		}, component.StabilityLevelStable),
	)
}

func TestCheckConsumeContractLogs(t *testing.T) {
	mr := &mockReceiver{}
	params := CheckConsumeContractParams{
		T:                    t,
		ExporterFactory:      newMockFactory(mr),
		Signal:               pipeline.SignalLogs,
		ExporterConfig:       nopConfig{},
		NumberOfTestElements: 10,
		ReceiverFactory:      newMockReceiverFactory(mr),
	}

	CheckConsumeContract(params)
}

func TestCheckConsumeContractMetrics(t *testing.T) {
	mr := &mockReceiver{}
	CheckConsumeContract(CheckConsumeContractParams{
		T:                    t,
		ExporterFactory:      newMockFactory(mr),
		Signal:               pipeline.SignalMetrics, // Change to the appropriate data type
		ExporterConfig:       nopConfig{},
		NumberOfTestElements: 10,
		ReceiverFactory:      newMockReceiverFactory(mr),
	})
}

func TestCheckConsumeContractTraces(t *testing.T) {
	mr := &mockReceiver{}
	CheckConsumeContract(CheckConsumeContractParams{
		T:                    t,
		ExporterFactory:      newMockFactory(mr),
		Signal:               pipeline.SignalTraces,
		ExporterConfig:       nopConfig{},
		NumberOfTestElements: 10,
		ReceiverFactory:      newMockReceiverFactory(mr),
	})
}
