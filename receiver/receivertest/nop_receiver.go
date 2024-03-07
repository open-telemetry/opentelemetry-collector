// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivertest // import "go.opentelemetry.io/collector/receiver/receivertest"

import (
	"context"

	"github.com/google/uuid"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

var defaultComponentType = component.MustNewType("nop")

// NewNopCreateSettings returns a new nop settings for Create*Receiver functions.
func NewNopCreateSettings() receiver.CreateSettings {
	return receiver.CreateSettings{
		ID:                component.NewIDWithName(defaultComponentType, uuid.NewString()),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

// NewNopFactory returns a receiver.Factory that constructs nop receivers supporting all data types.
func NewNopFactory() receiver.Factory {
	return receiver.NewFactory(
		defaultComponentType,
		func() component.Config { return &nopConfig{} },
		receiver.WithTraces(createTraces, component.StabilityLevelStable),
		receiver.WithMetrics(createMetrics, component.StabilityLevelStable),
		receiver.WithLogs(createLogs, component.StabilityLevelStable))
}

// NewNopFactoryForType returns a receiver.Factory that constructs nop receivers supporting only the
// given data type.
func NewNopFactoryForType(dataType component.DataType) receiver.Factory {
	var factoryOpt receiver.FactoryOption
	switch dataType {
	case component.DataTypeTraces:
		factoryOpt = receiver.WithTraces(createTraces, component.StabilityLevelStable)
	case component.DataTypeMetrics:
		factoryOpt = receiver.WithMetrics(createMetrics, component.StabilityLevelStable)
	case component.DataTypeLogs:
		factoryOpt = receiver.WithLogs(createLogs, component.StabilityLevelStable)
	default:
		panic("unsupported data type for creating nop receiver factory: " + dataType.String())
	}

	componentType := component.MustNewType(defaultComponentType.String() + "_" + dataType.String())
	return receiver.NewFactory(componentType, func() component.Config { return &nopConfig{} }, factoryOpt)
}

type nopConfig struct{}

func createTraces(context.Context, receiver.CreateSettings, component.Config, consumer.Traces) (receiver.Traces, error) {
	return nopInstance, nil
}

func createMetrics(context.Context, receiver.CreateSettings, component.Config, consumer.Metrics) (receiver.Metrics, error) {
	return nopInstance, nil
}

func createLogs(context.Context, receiver.CreateSettings, component.Config, consumer.Logs) (receiver.Logs, error) {
	return nopInstance, nil
}

var nopInstance = &nopReceiver{}

// nopReceiver acts as a receiver for testing purposes.
type nopReceiver struct {
	component.StartFunc
	component.ShutdownFunc
}

// NewNopBuilder returns a receiver.Builder that constructs nop receivers.
func NewNopBuilder() *receiver.Builder {
	nopFactory := NewNopFactory()
	return receiver.NewBuilder(
		map[component.ID]component.Config{component.NewID(defaultComponentType): nopFactory.CreateDefaultConfig()},
		map[component.Type]receiver.Factory{defaultComponentType: nopFactory})
}
