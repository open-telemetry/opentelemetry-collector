// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivertest // import "go.opentelemetry.io/collector/receiver/receivertest"

import (
	"context"

	"github.com/google/uuid"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/xreceiver"
)

var defaultComponentType = component.MustNewType("nop")

// NewNopSettings returns a new nop settings for Create*Receiver functions.
func NewNopSettings() receiver.Settings {
	return receiver.Settings{
		ID:                component.NewIDWithName(defaultComponentType, uuid.NewString()),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

// NewNopFactory returns a receiver.Factory that constructs nop receivers supporting all data types.
func NewNopFactory() receiver.Factory {
	return xreceiver.NewFactory(
		defaultComponentType,
		func() component.Config { return &nopConfig{} },
		xreceiver.WithTraces(createTraces, component.StabilityLevelStable),
		xreceiver.WithMetrics(createMetrics, component.StabilityLevelStable),
		xreceiver.WithLogs(createLogs, component.StabilityLevelStable),
		xreceiver.WithProfiles(createProfiles, component.StabilityLevelAlpha),
	)
}

// Deprecated: [v0.117.0] use NewNopFactory or create own factory.
func NewNopFactoryForType(signal pipeline.Signal) receiver.Factory {
	var factoryOpt receiver.FactoryOption
	switch signal {
	case pipeline.SignalTraces:
		factoryOpt = receiver.WithTraces(createTraces, component.StabilityLevelStable)
	case pipeline.SignalMetrics:
		factoryOpt = receiver.WithMetrics(createMetrics, component.StabilityLevelStable)
	case pipeline.SignalLogs:
		factoryOpt = receiver.WithLogs(createLogs, component.StabilityLevelStable)
	default:
		panic("unsupported data type for creating nop receiver factory: " + signal.String())
	}

	componentType := component.MustNewType(defaultComponentType.String() + "_" + signal.String())
	return receiver.NewFactory(componentType, func() component.Config { return &nopConfig{} }, factoryOpt)
}

type nopConfig struct{}

func createTraces(context.Context, receiver.Settings, component.Config, consumer.Traces) (receiver.Traces, error) {
	return nopInstance, nil
}

func createMetrics(context.Context, receiver.Settings, component.Config, consumer.Metrics) (receiver.Metrics, error) {
	return nopInstance, nil
}

func createLogs(context.Context, receiver.Settings, component.Config, consumer.Logs) (receiver.Logs, error) {
	return nopInstance, nil
}

func createProfiles(context.Context, receiver.Settings, component.Config, xconsumer.Profiles) (xreceiver.Profiles, error) {
	return nopInstance, nil
}

var nopInstance = &nopReceiver{}

// nopReceiver acts as a receiver for testing purposes.
type nopReceiver struct {
	component.StartFunc
	component.ShutdownFunc
}
