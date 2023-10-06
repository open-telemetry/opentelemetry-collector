// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpexporter // import "go.opentelemetry.io/collector/exporter/otlpexporter"

import (
	"context"
	"errors"
	"go.opentelemetry.io/collector/component"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

type mockOtlpReceiver struct {
	mockConsumer    *exportertest.MockConsumer
	traceReceiver   receiver.Traces
	metricsReceiver receiver.Metrics
	logReceiver     receiver.Logs
}

func newOTLPDataReceiver(mockConsumer *exportertest.MockConsumer) *mockOtlpReceiver {
	return &mockOtlpReceiver{
		mockConsumer: mockConsumer,
	}
}

func (bor *mockOtlpReceiver) Start(ctx context.Context, _ component.Host) error {
	factory := otlpreceiver.NewFactory()
	cfg := factory.CreateDefaultConfig().(*otlpreceiver.Config)
	cfg.HTTP = nil
	var err error
	set := receivertest.NewNopCreateSettings()
	if bor.traceReceiver, err = factory.CreateTracesReceiver(ctx, set, cfg, bor.mockConsumer); err != nil {
		return err
	}
	if bor.metricsReceiver, err = factory.CreateMetricsReceiver(ctx, set, cfg, bor.mockConsumer); err != nil {
		return err
	}
	if bor.logReceiver, err = factory.CreateLogsReceiver(ctx, set, cfg, bor.mockConsumer); err != nil {
		return err
	}

	if err = bor.traceReceiver.Start(ctx, componenttest.NewNopHost()); err != nil {
		return err
	}
	if err = bor.metricsReceiver.Start(ctx, componenttest.NewNopHost()); err != nil {
		return err
	}
	return bor.logReceiver.Start(ctx, componenttest.NewNopHost())
}

func (bor *mockOtlpReceiver) Shutdown(ctx context.Context) error {
	err := bor.traceReceiver.Shutdown(ctx)
	err = errors.Join(err, bor.metricsReceiver.Shutdown(ctx))
	return errors.Join(err, bor.logReceiver.Shutdown(ctx))
}
