// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testcomponents // import "go.opentelemetry.io/collector/service/internal/testcomponents"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterprofiles"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var testType = component.MustNewType("exampleexporter")

const stability = component.StabilityLevelDevelopment

// ExampleExporterFactory is factory for ExampleExporter.
var ExampleExporterFactory = exporter.NewFactory(
	testType,
	createExporterDefaultConfig,
	exporter.WithTraces(createTracesExporter, stability),
	exporter.WithMetrics(createMetricsExporter, stability),
	exporter.WithLogs(createLogsExporter, stability),
	exporterprofiles.WithProfiles(createProfilesExporter, stability),
)

func createExporterDefaultConfig() component.Config {
	return &struct{}{}
}

func createTracesExporter(context.Context, exporter.Settings, component.Config) (exporter.Traces, error) {
	return &ExampleExporter{}, nil
}

func createMetricsExporter(context.Context, exporter.Settings, component.Config) (exporter.Metrics, error) {
	return &ExampleExporter{}, nil
}

func createLogsExporter(context.Context, exporter.Settings, component.Config) (exporter.Logs, error) {
	return &ExampleExporter{}, nil
}
func createProfilesExporter(context.Context, exporter.Settings, component.Config) (exporterprofiles.Profiles, error) {
	return &ExampleExporter{}, nil
}

// ExampleExporter stores consumed traces, metrics, logs and profiles for testing purposes.
type ExampleExporter struct {
	componentState
	Traces   []ptrace.Traces
	Metrics  []pmetric.Metrics
	Logs     []plog.Logs
	Profiles []pprofile.Profiles
}

// ConsumeTraces receives ptrace.Traces for processing by the consumer.Traces.
func (exp *ExampleExporter) ConsumeTraces(_ context.Context, td ptrace.Traces) error {
	exp.Traces = append(exp.Traces, td)
	return nil
}

// ConsumeMetrics receives pmetric.Metrics for processing by the Metrics.
func (exp *ExampleExporter) ConsumeMetrics(_ context.Context, md pmetric.Metrics) error {
	exp.Metrics = append(exp.Metrics, md)
	return nil
}

// ConsumeLogs receives plog.Logs for processing by the Logs.
func (exp *ExampleExporter) ConsumeLogs(_ context.Context, ld plog.Logs) error {
	exp.Logs = append(exp.Logs, ld)
	return nil
}

// ConsumeProfiles receives pprofile.Profiles for processing by the consumerprofiles.Profiles.
func (exp *ExampleExporter) ConsumeProfiles(_ context.Context, td pprofile.Profiles) error {
	exp.Profiles = append(exp.Profiles, td)
	return nil
}

func (exp *ExampleExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}
