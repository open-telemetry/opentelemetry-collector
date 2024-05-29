// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package nopreceiver // import "go.opentelemetry.io/collector/receiver/nopreceiver"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumerlogs"
	"go.opentelemetry.io/collector/consumer/consumermetrics"
	"go.opentelemetry.io/collector/consumer/consumertraces"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/nopreceiver/internal/metadata"
)

// NewFactory returns a receiver.Factory that constructs nop receivers.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		func() component.Config { return &struct{}{} },
		receiver.WithTraces(createTraces, metadata.TracesStability),
		receiver.WithMetrics(createMetrics, metadata.MetricsStability),
		receiver.WithLogs(createLogs, metadata.LogsStability))
}

func createTraces(context.Context, receiver.CreateSettings, component.Config, consumertraces.Traces) (receiver.Traces, error) {
	return nopInstance, nil
}

func createMetrics(context.Context, receiver.CreateSettings, component.Config, consumermetrics.Metrics) (receiver.Metrics, error) {
	return nopInstance, nil
}

func createLogs(context.Context, receiver.CreateSettings, component.Config, consumerlogs.Logs) (receiver.Logs, error) {
	return nopInstance, nil
}

var nopInstance = &nopReceiver{}

type nopReceiver struct {
	component.StartFunc
	component.ShutdownFunc
}
