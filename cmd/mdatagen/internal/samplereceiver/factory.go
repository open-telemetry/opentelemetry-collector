// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package samplereceiver // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/samplereceiver"

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal/samplereceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

// NewFactory returns a receiver.Factory for sample receiver.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		func() component.Config { return &struct{}{} },
		receiver.WithTraces(createTraces, metadata.TracesStability),
		receiver.WithMetrics(createMetrics, metadata.MetricsStability),
		receiver.WithLogs(createLogs, metadata.LogsStability))
}

func createTraces(context.Context, receiver.Settings, component.Config, consumer.Traces) (receiver.Traces, error) {
	return nopInstance, nil
}

func createMetrics(ctx context.Context, set receiver.Settings, _ component.Config, _ consumer.Metrics) (receiver.Metrics, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	err = telemetryBuilder.RegisterProcessRuntimeTotalAllocBytesCallback(func(_ context.Context, observer metric.Int64Observer) error {
		observer.Observe(2)
		return nil
	})
	if err != nil {
		return nil, err
	}

	telemetryBuilder.BatchSizeTriggerSend.Add(ctx, 1)
	return nopReceiver{telemetryBuilder: telemetryBuilder}, nil
}

func createLogs(context.Context, receiver.Settings, component.Config, consumer.Logs) (receiver.Logs, error) {
	return nopInstance, nil
}

var nopInstance = &nopReceiver{}

type nopReceiver struct {
	component.StartFunc
	telemetryBuilder *metadata.TelemetryBuilder
}

func (r nopReceiver) initOptionalMetric() {
	_ = r.telemetryBuilder.RegisterQueueLengthCallback(func(_ context.Context, observer metric.Int64Observer) error {
		observer.Observe(3)
		return nil
	})
}

// Shutdown shuts down the component.
func (r nopReceiver) Shutdown(context.Context) error {
	if r.telemetryBuilder != nil {
		r.telemetryBuilder.Shutdown()
	}
	return nil
}
