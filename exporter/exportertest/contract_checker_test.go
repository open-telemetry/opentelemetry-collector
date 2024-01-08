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

// mockExporterFactory is a factory to create exporters sending data to the mockReceiver.
type mockExporterFactory struct {
	mr *mockReceiver
	component.StartFunc
	component.ShutdownFunc
}

func (mef *mockExporterFactory) createMockTracesExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
) (exporter.Traces, error) {
	return exporterhelper.NewTracesExporter(ctx, set, cfg,
		mef.mr.Traces.ConsumeTraces,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithRetry(retryConfig),
	)
}

func (mef *mockExporterFactory) createMockMetricsExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
) (exporter.Metrics, error) {
	return exporterhelper.NewMetricsExporter(ctx, set, cfg,
		mef.mr.Metrics.ConsumeMetrics,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithRetry(retryConfig),
	)
}

func (mef *mockExporterFactory) createMockLogsExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
) (exporter.Logs, error) {
	return exporterhelper.NewLogsExporter(ctx, set, cfg,
		mef.mr.Logs.ConsumeLogs,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithRetry(retryConfig),
	)
}

func newMockExporterFactory(mr *mockReceiver) exporter.Factory {
	mef := &mockExporterFactory{mr: mr}
	return exporter.NewFactory(
		"pass_through_exporter",
		func() component.Config { return &nopConfig{} },
		exporter.WithTraces(mef.createMockTracesExporter, component.StabilityLevelBeta),
		exporter.WithMetrics(mef.createMockMetricsExporter, component.StabilityLevelBeta),
		exporter.WithLogs(mef.createMockLogsExporter, component.StabilityLevelBeta),
	)
}

func newMockReceiverFactory(mr *mockReceiver) receiver.Factory {
	return receiver.NewFactory("pass_through_receiver",
		func() component.Config { return &nopConfig{} },
		receiver.WithTraces(func(_ context.Context, _ receiver.CreateSettings, _ component.Config, c consumer.Traces) (receiver.Traces, error) {
			mr.Traces = c
			return mr, nil
		}, component.StabilityLevelStable),
		receiver.WithMetrics(func(_ context.Context, _ receiver.CreateSettings, _ component.Config, c consumer.Metrics) (receiver.Metrics, error) {
			mr.Metrics = c
			return mr, nil
		}, component.StabilityLevelStable),
		receiver.WithLogs(func(_ context.Context, _ receiver.CreateSettings, _ component.Config, c consumer.Logs) (receiver.Logs, error) {
			mr.Logs = c
			return mr, nil
		}, component.StabilityLevelStable),
	)
}

func TestCheckConsumeContractLogs(t *testing.T) {
	mr := &mockReceiver{}
	params := CheckConsumeContractParams{
		T:                    t,
		ExporterFactory:      newMockExporterFactory(mr),
		DataType:             component.DataTypeLogs,
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
		ExporterFactory:      newMockExporterFactory(mr),
		DataType:             component.DataTypeMetrics, // Change to the appropriate data type
		ExporterConfig:       nopConfig{},
		NumberOfTestElements: 10,
		ReceiverFactory:      newMockReceiverFactory(mr),
	})
}

func TestCheckConsumeContractTraces(t *testing.T) {
	mr := &mockReceiver{}
	CheckConsumeContract(CheckConsumeContractParams{
		T:                    t,
		ExporterFactory:      newMockExporterFactory(mr),
		DataType:             component.DataTypeTraces,
		ExporterConfig:       nopConfig{},
		NumberOfTestElements: 10,
		ReceiverFactory:      newMockReceiverFactory(mr),
	})
}
