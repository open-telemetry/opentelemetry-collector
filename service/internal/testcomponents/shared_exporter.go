// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testcomponents // import "go.opentelemetry.io/collector/service/internal/testcomponents"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/xexporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var sharedExporterType = component.MustNewType("sharedexporter")

// SharedExporterFactory is factory for SharedExporter.
var SharedExporterFactory = xexporter.NewFactory(
	sharedExporterType,
	createSharedExporterDefaultConfig,
	xexporter.WithTraces(createSharedTracesExporter, component.StabilityLevelDevelopment),
	xexporter.WithMetrics(createSharedMetricsExporter, component.StabilityLevelDevelopment),
	xexporter.WithLogs(createSharedLogsExporter, component.StabilityLevelDevelopment),
	xexporter.WithProfiles(createSharedProfilesExporter, component.StabilityLevelDevelopment),
	xexporter.WithSharedInstance(),
)

func createSharedExporterDefaultConfig() component.Config {
	return &struct{}{}
}

func createSharedTracesExporter(
	_ context.Context,
	_ exporter.Settings,
	cfg component.Config,
) (exporter.Traces, error) {
	return createSharedExporter(cfg), nil
}

func createSharedMetricsExporter(
	_ context.Context,
	_ exporter.Settings,
	cfg component.Config,
) (exporter.Metrics, error) {
	return createSharedExporter(cfg), nil
}

func createSharedLogsExporter(
	_ context.Context,
	_ exporter.Settings,
	cfg component.Config,
) (exporter.Logs, error) {
	return createSharedExporter(cfg), nil
}

func createSharedProfilesExporter(
	_ context.Context,
	_ exporter.Settings,
	cfg component.Config,
) (xexporter.Profiles, error) {
	return createSharedExporter(cfg), nil
}

func createSharedExporter(cfg component.Config) *SharedExporter {
	// There must be one receiver for all data types. We maintain a map of
	// receivers per config.

	// Check to see if there is already a receiver for this config.
	sr, ok := sharedExporters[cfg]
	if !ok {
		sr = &SharedExporter{}
		// Remember the receiver in the map
		sharedExporters[cfg] = sr
	}

	return sr
}

// SharedExporter stores consumed traces, metrics, logs and profiles for testing purposes.
type SharedExporter struct {
	componentState
	Traces   []ptrace.Traces
	Metrics  []pmetric.Metrics
	Logs     []plog.Logs
	Profiles []pprofile.Profiles
}

// ConsumeTraces receives ptrace.Traces for processing by the consumer.Traces.
func (exp *SharedExporter) ConsumeTraces(_ context.Context, td ptrace.Traces) error {
	exp.Traces = append(exp.Traces, td)
	return nil
}

// ConsumeMetrics receives pmetric.Metrics for processing by the Metrics.
func (exp *SharedExporter) ConsumeMetrics(_ context.Context, md pmetric.Metrics) error {
	exp.Metrics = append(exp.Metrics, md)
	return nil
}

// ConsumeLogs receives plog.Logs for processing by the Logs.
func (exp *SharedExporter) ConsumeLogs(_ context.Context, ld plog.Logs) error {
	exp.Logs = append(exp.Logs, ld)
	return nil
}

// ConsumeProfiles receives pprofile.Profiles for processing by the xconsumer.Profiles.
func (exp *SharedExporter) ConsumeProfiles(_ context.Context, td pprofile.Profiles) error {
	exp.Profiles = append(exp.Profiles, td)
	return nil
}

func (exp *SharedExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// This is the map of already created shared exporters for particular IDs.
// We maintain this map because the receiver.Factory is asked trace and metric exporters separately
// when it gets CreateTraces() and CreateMetrics() but they must not
// create separate objects, they must use one Receiver object per configuration.
var sharedExporters = map[component.Config]*SharedExporter{}
