// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivertest // import "go.opentelemetry.io/collector/receiver/receivertest"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

var nopLogsComponentType = component.MustNewType("nop_logs")

// NewNopLogsCreateSettings returns a new nop settings for Create*Receiver functions.
func NewNopLogsCreateSettings() receiver.CreateSettings {
	return receiver.CreateSettings{
		ID:                component.NewID(nopLogsComponentType),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

// NewNopLogsFactory returns a receiver.Factory that constructs nop logs receivers.
func NewNopLogsFactory() receiver.Factory {
	createLogs := func(context.Context, receiver.CreateSettings, component.Config, consumer.Logs) (receiver.Logs, error) {
		return nopInstance, nil
	}

	return receiver.NewFactory(
		nopLogsComponentType,
		func() component.Config { return &nopLogsConfig{} },
		receiver.WithLogs(createLogs, component.StabilityLevelStable))
}

type nopLogsConfig struct{}

var nopLogsInstance = &nopLogsReceiver{}

// nopLogsReceiver acts as a receiver for testing purposes.
type nopLogsReceiver struct {
	component.StartFunc
	component.ShutdownFunc
}

// NewNopLogsBuilder returns a receiver.Builder that constructs nop receivers.
func NewNopLogsBuilder() *receiver.Builder {
	nopFactory := NewNopLogsFactory()
	return receiver.NewBuilder(
		map[component.ID]component.Config{component.NewID(componentType): nopFactory.CreateDefaultConfig()},
		map[component.Type]receiver.Factory{componentType: nopFactory})
}
