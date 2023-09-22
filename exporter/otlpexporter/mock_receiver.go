// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpexporter // import "go.opentelemetry.io/collector/exporter/otlpexporter"

import (
	"context"
	"errors"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

type mockOtlpReceiver struct {
	mockConsumer    exportertest.MockConsumer
	traceReceiver   receiver.Traces
	metricsReceiver receiver.Metrics
	logReceiver     receiver.Logs
}

func newOTLPDataReceiver(mockConsumer exportertest.MockConsumer) *mockOtlpReceiver {
	return &mockOtlpReceiver{
		mockConsumer: mockConsumer,
	}
}

func (bor *mockOtlpReceiver) Start() error {
	factory := otlpreceiver.NewFactory()
	cfg := factory.CreateDefaultConfig().(*otlpreceiver.Config)
	cfg.HTTP = nil
	var err error
	set := receivertest.NewNopCreateSettings()
	if bor.traceReceiver, err = factory.CreateTracesReceiver(context.Background(), set, cfg, bor.mockConsumer); err != nil {
		return err
	}
	if bor.metricsReceiver, err = factory.CreateMetricsReceiver(context.Background(), set, cfg, bor.mockConsumer); err != nil {
		return err
	}
	if bor.logReceiver, err = factory.CreateLogsReceiver(context.Background(), set, cfg, bor.mockConsumer); err != nil {
		return err
	}

	if err = bor.traceReceiver.Start(context.Background(), componenttest.NewNopHost()); err != nil {
		return err
	}
	if err = bor.metricsReceiver.Start(context.Background(), componenttest.NewNopHost()); err != nil {
		return err
	}
	return bor.logReceiver.Start(context.Background(), componenttest.NewNopHost())
}

func (bor *mockOtlpReceiver) Stop() error {
	bor.mockConsumer.Clear()
	err := bor.traceReceiver.Shutdown(context.Background())
	err = errors.Join(err, bor.metricsReceiver.Shutdown(context.Background()))
	return errors.Join(err, bor.logReceiver.Shutdown(context.Background()))
}

func (bor *mockOtlpReceiver) RequestCounter() exportertest.RequestCounter {
	return bor.mockConsumer.RequestCounter()
}
