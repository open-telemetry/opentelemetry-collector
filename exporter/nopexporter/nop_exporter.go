// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package nopexporter // import "go.opentelemetry.io/collector/exporter/nopexporter"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/nopexporter/internal/metadata"
)

// NewFactory returns an exporter.Factory that constructs nop exporters.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		metadata.Type,
		func() component.Config { return &struct{}{} },
		exporter.WithTraces(createTraces, metadata.TracesStability),
		exporter.WithMetrics(createMetrics, metadata.MetricsStability),
		exporter.WithLogs(createLogs, metadata.LogsStability),
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

var nopInstance = &nop{
	Consumer: consumertest.NewNop(),
}

type nop struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}
