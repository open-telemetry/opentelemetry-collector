// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exportertest // import "go.opentelemetry.io/collector/exporter/exportertest"

import (
	"context"

	"github.com/google/uuid"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterprofiles"
)

var nopType = component.MustNewType("nop")

// NewNopSettings returns a new nop settings for Create* functions.
func NewNopSettings() exporter.Settings {
	return exporter.Settings{
		ID:                component.NewIDWithName(nopType, uuid.NewString()),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

// NewNopFactory returns an exporter.Factory that constructs nop exporters.
func NewNopFactory() exporter.Factory {
	return exporterprofiles.NewFactory(
		nopType,
		func() component.Config { return &nopConfig{} },
		exporterprofiles.WithTraces(createTraces, component.StabilityLevelStable),
		exporterprofiles.WithMetrics(createMetrics, component.StabilityLevelStable),
		exporterprofiles.WithLogs(createLogs, component.StabilityLevelStable),
		exporterprofiles.WithProfiles(createProfiles, component.StabilityLevelAlpha),
	)
}

func createTraces(context.Context, exporter.Settings, component.Config) (exporter.Traces, error) {
	return nopInstance, nil
}

func createMetrics(context.Context, exporter.Settings, component.Config) (exporter.Metrics, error) {
	return nopInstance, nil
}

func createLogs(context.Context, exporter.Settings, component.Config) (exporter.Logs, error) {
	return nopInstance, nil
}

func createProfiles(context.Context, exporter.Settings, component.Config) (exporterprofiles.Profiles, error) {
	return nopInstance, nil
}

type nopConfig struct{}

var nopInstance = &nop{
	Consumer: consumertest.NewNop(),
}

// nop stores consumed traces, metrics, logs and profiles for testing purposes.
type nop struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}
