// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testcomponents // import "go.opentelemetry.io/collector/service/internal/testcomponents"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/xreceiver"
)

var receiverType = component.MustNewType("examplereceiver")

// ExampleReceiverFactory is factory for ExampleReceiver.
var ExampleReceiverFactory = xreceiver.NewFactory(
	receiverType,
	createReceiverDefaultConfig,
	xreceiver.WithTraces(createTracesReceiver, component.StabilityLevelDevelopment),
	xreceiver.WithMetrics(createMetricsReceiver, component.StabilityLevelDevelopment),
	xreceiver.WithLogs(createLogsReceiver, component.StabilityLevelDevelopment),
	xreceiver.WithProfiles(createProfilesReceiver, component.StabilityLevelDevelopment),
)

func createReceiverDefaultConfig() component.Config {
	return &struct{}{}
}

// createTraces creates a receiver.Traces based on this config.
func createTracesReceiver(
	_ context.Context,
	_ receiver.Settings,
	_ component.Config,
	nextConsumer consumer.Traces,
) (receiver.Traces, error) {
	return &ExampleReceiver{ConsumeTracesFunc: nextConsumer.ConsumeTraces}, nil
}

// createMetrics creates a receiver.Metrics based on this config.
func createMetricsReceiver(
	_ context.Context,
	_ receiver.Settings,
	_ component.Config,
	nextConsumer consumer.Metrics,
) (receiver.Metrics, error) {
	return &ExampleReceiver{ConsumeMetricsFunc: nextConsumer.ConsumeMetrics}, nil
}

// createLogs creates a receiver.Logs based on this config.
func createLogsReceiver(
	_ context.Context,
	_ receiver.Settings,
	_ component.Config,
	nextConsumer consumer.Logs,
) (receiver.Logs, error) {
	return &ExampleReceiver{ConsumeLogsFunc: nextConsumer.ConsumeLogs}, nil
}

// createProfiles creates a receiver.Profiles based on this config.
func createProfilesReceiver(
	_ context.Context,
	_ receiver.Settings,
	_ component.Config,
	nextConsumer xconsumer.Profiles,
) (xreceiver.Profiles, error) {
	return &ExampleReceiver{ConsumeProfilesFunc: nextConsumer.ConsumeProfiles}, nil
}

// ExampleReceiver allows producing traces, metrics, logs and profiles for testing purposes.
type ExampleReceiver struct {
	componentState
	consumer.ConsumeTracesFunc
	consumer.ConsumeMetricsFunc
	consumer.ConsumeLogsFunc
	xconsumer.ConsumeProfilesFunc
}
