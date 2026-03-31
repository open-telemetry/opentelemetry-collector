// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package nopexporter // import "go.opentelemetry.io/collector/exporter/nopexporter"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/nopexporter/internal/metadata"
	"go.opentelemetry.io/collector/exporter/xexporter"
)

// NewFactory returns an exporter.Factory that constructs nop exporters.
func NewFactory() exporter.Factory {
	return xexporter.NewFactory(
		metadata.Type,
		func() component.Config { return &struct{}{} },
		xexporter.WithTraces(createTraces, metadata.TracesStability),
		xexporter.WithMetrics(createMetrics, metadata.MetricsStability),
		xexporter.WithLogs(createLogs, metadata.LogsStability),
		xexporter.WithProfiles(createProfiles, metadata.ProfilesStability),
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

func createProfiles(context.Context, exporter.Settings, component.Config) (xexporter.Profiles, error) {
	return nopInstance, nil
}

var nopInstance = &nop{
	Consumer: consumertest.NewNop(),
}

type nop struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}
