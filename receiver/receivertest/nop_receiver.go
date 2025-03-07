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
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/xreceiver"
)

var NopType = component.MustNewType("nop")

// NewNopSettings returns a new nop settings for Create*Receiver functions with the given type.
func NewNopSettings(typ component.Type) receiver.Settings {
	return receiver.Settings{
		ID:                component.NewIDWithName(typ, uuid.NewString()),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

// NewNopFactory returns a receiver.Factory that constructs nop receivers supporting all data types.
func NewNopFactory() receiver.Factory {
	return xreceiver.NewFactory(
		NopType,
		func() component.Config { return &nopConfig{} },
		xreceiver.WithTraces(createTraces, component.StabilityLevelStable),
		xreceiver.WithMetrics(createMetrics, component.StabilityLevelStable),
		xreceiver.WithLogs(createLogs, component.StabilityLevelStable),
		xreceiver.WithProfiles(createProfiles, component.StabilityLevelAlpha),
	)
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
