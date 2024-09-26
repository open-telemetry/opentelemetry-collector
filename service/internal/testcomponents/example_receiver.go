// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testcomponents // import "go.opentelemetry.io/collector/service/internal/testcomponents"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverprofiles"
)

var receiverType = component.MustNewType("examplereceiver")

// ExampleReceiverFactory is factory for ExampleReceiver.
var ExampleReceiverFactory = receiverprofiles.NewFactory(
	receiverType,
	createReceiverDefaultConfig,
	receiverprofiles.WithTraces(createTracesReceiver, component.StabilityLevelDevelopment),
	receiverprofiles.WithMetrics(createMetricsReceiver, component.StabilityLevelDevelopment),
	receiverprofiles.WithLogs(createLogsReceiver, component.StabilityLevelDevelopment),
	receiverprofiles.WithProfiles(createProfilesReceiver, component.StabilityLevelDevelopment),
)

func createReceiverDefaultConfig() component.Config {
	return &struct{}{}
}

// createTracesReceiver creates a trace receiver based on this config.
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

// createMetricsReceiver creates a metrics receiver based on this config.
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

// createProfilesReceiver creates a trace receiver based on this config.
func createProfilesReceiver(
	_ context.Context,
	_ receiver.Settings,
	cfg component.Config,
	nextConsumer consumerprofiles.Profiles,
) (receiverprofiles.Profiles, error) {
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
	consumerprofiles.ConsumeProfilesFunc
}

// This is the map of already created example receivers for particular configurations.
// We maintain this map because the ReceiverFactory is asked trace and metric receivers separately
// when it gets CreateTracesReceiver() and CreateMetricsReceiver() but they must not
// create separate objects, they must use one Receiver object per configuration.
var exampleReceivers = map[component.Config]*ExampleReceiver{}
