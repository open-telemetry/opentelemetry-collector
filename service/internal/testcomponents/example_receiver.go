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
	cfg component.Config,
	nextConsumer consumer.Traces,
) (receiver.Traces, error) {
	tr := createReceiver(cfg)
	tr.ConsumeTracesFunc = nextConsumer.ConsumeTraces
	return tr, nil
}

// createMetrics creates a receiver.Metrics based on this config.
func createMetricsReceiver(
	_ context.Context,
	_ receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (receiver.Metrics, error) {
	mr := createReceiver(cfg)
	mr.ConsumeMetricsFunc = nextConsumer.ConsumeMetrics
	return mr, nil
}

// createLogs creates a receiver.Logs based on this config.
func createLogsReceiver(
	_ context.Context,
	_ receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (receiver.Logs, error) {
	lr := createReceiver(cfg)
	lr.ConsumeLogsFunc = nextConsumer.ConsumeLogs
	return lr, nil
}

// createProfiles creates a receiver.Profiles based on this config.
func createProfilesReceiver(
	_ context.Context,
	_ receiver.Settings,
	cfg component.Config,
	nextConsumer xconsumer.Profiles,
) (xreceiver.Profiles, error) {
	tr := createReceiver(cfg)
	tr.ConsumeProfilesFunc = nextConsumer.ConsumeProfiles
	return tr, nil
}

func createReceiver(cfg component.Config) *ExampleReceiver {
	// There must be one receiver for all data types. We maintain a map of
	// receivers per config.

	// Check to see if there is already a receiver for this config.
	er, ok := exampleReceivers[cfg]
	if !ok {
		er = &ExampleReceiver{}
		// Remember the receiver in the map
		exampleReceivers[cfg] = er
	}

	return er
}

// ExampleReceiver allows producing traces, metrics, logs and profiles for testing purposes.
type ExampleReceiver struct {
	componentState
	consumer.ConsumeTracesFunc
	consumer.ConsumeMetricsFunc
	consumer.ConsumeLogsFunc
	xconsumer.ConsumeProfilesFunc
}

// This is the map of already created example receivers for particular configurations.
// We maintain this map because the receiver.Factory is asked trace and metric receivers separately
// when it gets CreateTraces() and CreateMetrics() but they must not
// create separate objects, they must use one Receiver object per configuration.
var exampleReceivers = map[component.Config]*ExampleReceiver{}
