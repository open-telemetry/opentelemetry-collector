// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package samplefactoryreceiver // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/samplefactoryreceiver"

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal/samplefactoryreceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/xreceiver"
	"go.opentelemetry.io/collector/service/hostcapabilities"
)

// NewFactory returns a receiver.Factory for sample receiver.
func NewFactory() xreceiver.Factory {
	return xreceiver.NewFactory(
		metadata.Type,
		func() component.Config { return &struct{}{} },
		xreceiver.WithTraces(createTraces, metadata.TracesStability),
		xreceiver.WithMetrics(createMetrics, metadata.MetricsStability),
		xreceiver.WithLogs(createLogs, metadata.LogsStability),
		xreceiver.WithProfiles(createProfiles, metadata.ProfilesStability),
	)
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

func createProfiles(context.Context, receiver.Settings, component.Config, xconsumer.Profiles) (xreceiver.Profiles, error) {
	return nopInstance, nil
}

var nopInstance = &nopReceiver{}

type nopReceiver struct {
	component.StartFunc
	telemetryBuilder *metadata.TelemetryBuilder
}

func (r nopReceiver) Start(_ context.Context, host component.Host) error {
	if _, ok := host.(hostcapabilities.ComponentFactory); !ok {
		return errors.New("host does not implement hostcapabilities.ComponentFactory")
	}

	return nil
}

// Shutdown shuts down the component.
func (r nopReceiver) Shutdown(context.Context) error {
	if r.telemetryBuilder != nil {
		r.telemetryBuilder.Shutdown()
	}
	return nil
}
