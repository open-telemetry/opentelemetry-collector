// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processortest // import "go.opentelemetry.io/collector/processor/processortest"

import (
	"context"

	"github.com/google/uuid"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorprofiles"
)

var nopType = component.MustNewType("nop")

// NewNopSettings returns a new nop settings for Create* functions.
func NewNopSettings() processor.Settings {
	return processor.Settings{
		ID:                component.NewIDWithName(nopType, uuid.NewString()),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

// NewNopFactory returns a component.ProcessorFactory that constructs nop processors.
func NewNopFactory() processor.Factory {
	return processorprofiles.NewFactory(
		nopType,
		func() component.Config { return &nopConfig{} },
		processorprofiles.WithTraces(createTraces, component.StabilityLevelStable),
		processorprofiles.WithMetrics(createMetrics, component.StabilityLevelStable),
		processorprofiles.WithLogs(createLogs, component.StabilityLevelStable),
		processorprofiles.WithProfiles(createProfiles, component.StabilityLevelAlpha),
	)
}

func createTraces(context.Context, processor.Settings, component.Config, consumer.Traces) (processor.Traces, error) {
	return nopInstance, nil
}

func createMetrics(context.Context, processor.Settings, component.Config, consumer.Metrics) (processor.Metrics, error) {
	return nopInstance, nil
}

func createLogs(context.Context, processor.Settings, component.Config, consumer.Logs) (processor.Logs, error) {
	return nopInstance, nil
}

func createProfiles(context.Context, processor.Settings, component.Config, consumerprofiles.Profiles) (processorprofiles.Profiles, error) {
	return nopInstance, nil
}

type nopConfig struct{}

var nopInstance = &nop{
	Consumer: consumertest.NewNop(),
}

// nop acts as a processor for testing purposes.
type nop struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}
