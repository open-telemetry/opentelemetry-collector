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

const typeStr = "nop"

// NewNopCreateSettings returns a new nop settings for Create* functions.
func NewNopCreateSettings() receiver.CreateSettings {
	return receiver.CreateSettings{
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

// NewNopFactory returns a receiver.Factory that constructs nop receivers.
func NewNopFactory() receiver.Factory {
	return receiver.NewFactory(
		"nop",
		func() component.Config { return &nopConfig{} },
		receiver.WithTraces(createTraces, component.StabilityLevelStable),
		receiver.WithMetrics(createMetrics, component.StabilityLevelStable),
		receiver.WithLogs(createLogs, component.StabilityLevelStable))
}

func createTraces(context.Context, receiver.CreateSettings, component.Config, consumer.Traces) (receiver.Traces, error) {
	return nopInstance, nil
}

func createMetrics(context.Context, receiver.CreateSettings, component.Config, consumer.Metrics) (receiver.Metrics, error) {
	return nopInstance, nil
}

func createLogs(context.Context, receiver.CreateSettings, component.Config, consumer.Logs) (receiver.Logs, error) {
	return nopInstance, nil
}

type nopConfig struct{}

var nopInstance = &nopReceiver{}

// nopReceiver stores consumed traces and metrics for testing purposes.
type nopReceiver struct {
	component.StartFunc
	component.ShutdownFunc
}

// NewNopBuilder returns a receiver.Builder that constructs nop receivers.
func NewNopBuilder() *receiver.Builder {
	nopFactory := NewNopFactory()
	return receiver.NewBuilder(
		map[component.ID]component.Config{component.NewID(typeStr): nopFactory.CreateDefaultConfig()},
		map[component.Type]receiver.Factory{typeStr: nopFactory})
}
