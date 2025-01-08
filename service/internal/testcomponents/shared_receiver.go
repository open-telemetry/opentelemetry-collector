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

var sharedReceiverType = component.MustNewType("sharedreceiver")

// SharedReceiverFactory is factory for SharedReceiver.
var SharedReceiverFactory = xreceiver.NewFactory(
	sharedReceiverType,
	createSharedReceiverDefaultConfig,
	xreceiver.WithTraces(createSharedTracesReceiver, component.StabilityLevelDevelopment),
	xreceiver.WithMetrics(createSharedMetricsReceiver, component.StabilityLevelDevelopment),
	xreceiver.WithLogs(createSharedLogsReceiver, component.StabilityLevelDevelopment),
	xreceiver.WithProfiles(createSharedProfilesReceiver, component.StabilityLevelDevelopment),
	xreceiver.WithSharedInstance(),
)

func createSharedReceiverDefaultConfig() component.Config {
	return &struct{}{}
}

func createSharedTracesReceiver(
	_ context.Context,
	_ receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (receiver.Traces, error) {
	r := createSharedReceiver(cfg)
	r.ConsumeTracesFunc = nextConsumer.ConsumeTraces
	return r, nil
}

func createSharedMetricsReceiver(
	_ context.Context,
	_ receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (receiver.Metrics, error) {
	r := createSharedReceiver(cfg)
	r.ConsumeMetricsFunc = nextConsumer.ConsumeMetrics
	return r, nil
}

func createSharedLogsReceiver(
	_ context.Context,
	_ receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (receiver.Logs, error) {
	r := createSharedReceiver(cfg)
	r.ConsumeLogsFunc = nextConsumer.ConsumeLogs
	return r, nil
}

func createSharedProfilesReceiver(
	_ context.Context,
	_ receiver.Settings,
	cfg component.Config,
	nextConsumer xconsumer.Profiles,
) (xreceiver.Profiles, error) {
	r := createSharedReceiver(cfg)
	r.ConsumeProfilesFunc = nextConsumer.ConsumeProfiles
	return r, nil
}

func createSharedReceiver(cfg component.Config) *SharedReceiver {
	// There must be one receiver for all data types. We maintain a map of
	// receivers per config.

	// Check to see if there is already a receiver for this config.
	sr, ok := sharedReceivers[cfg]
	if !ok {
		sr = &SharedReceiver{}
		// Remember the receiver in the map
		sharedReceivers[cfg] = sr
	}

	return sr
}

// SharedReceiver allows producing traces, metrics, logs and profiles for testing purposes.
type SharedReceiver struct {
	componentState
	consumer.ConsumeTracesFunc
	consumer.ConsumeMetricsFunc
	consumer.ConsumeLogsFunc
	xconsumer.ConsumeProfilesFunc
}

// This is the map of already created shared receivers for particular configurations.
// We maintain this map because the receiver.Factory is asked trace and metric receivers separately
// when it gets CreateTraces() and CreateMetrics() but they must not
// create separate objects, they must use one Receiver object per configuration.
var sharedReceivers = map[component.Config]*SharedReceiver{}
