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

// NewNopSettings returns a new nop settings for Create*Exporter functions.
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
		exporterprofiles.WithTraces(createTracesExporter, component.StabilityLevelStable),
		exporterprofiles.WithMetrics(createMetricsExporter, component.StabilityLevelStable),
		exporterprofiles.WithLogs(createLogsExporter, component.StabilityLevelStable),
		exporterprofiles.WithProfiles(createProfilesExporter, component.StabilityLevelAlpha),
	)
}

func createTracesExporter(context.Context, exporter.Settings, component.Config) (exporter.Traces, error) {
	return nopInstance, nil
}

func createMetricsExporter(context.Context, exporter.Settings, component.Config) (exporter.Metrics, error) {
	return nopInstance, nil
}

func createLogsExporter(context.Context, exporter.Settings, component.Config) (exporter.Logs, error) {
	return nopInstance, nil
}

func createProfilesExporter(context.Context, exporter.Settings, component.Config) (exporterprofiles.Profiles, error) {
	return nopInstance, nil
}

type nopConfig struct{}

var nopInstance = &nopExporter{
	Consumer: consumertest.NewNop(),
}

// nopExporter stores consumed traces, metrics, logs and profiles for testing purposes.
type nopExporter struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}
