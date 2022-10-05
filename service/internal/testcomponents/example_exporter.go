// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testcomponents // import "go.opentelemetry.io/collector/service/internal/testcomponents"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	typeStr   = "exampleexporter"
	stability = component.StabilityLevelInDevelopment
)

// ExampleExporterConfig config for ExampleExporter.
type ExampleExporterConfig struct {
	config.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
}

// ExampleExporterFactory is factory for ExampleExporter.
var ExampleExporterFactory = component.NewExporterFactory(
	typeStr,
	createExporterDefaultConfig,
	component.WithTracesExporter(createTracesExporter, stability),
	component.WithMetricsExporter(createMetricsExporter, stability),
	component.WithLogsExporter(createLogsExporter, stability),
)

func createExporterDefaultConfig() config.Exporter {
	return &ExampleExporterConfig{
		ExporterSettings: config.NewExporterSettings(config.NewComponentID(typeStr)),
	}
}

func createTracesExporter(context.Context, component.ExporterCreateSettings, config.Exporter) (component.TracesExporter, error) {
	return &ExampleExporter{}, nil
}

func createMetricsExporter(context.Context, component.ExporterCreateSettings, config.Exporter) (component.MetricsExporter, error) {
	return &ExampleExporter{}, nil
}

func createLogsExporter(context.Context, component.ExporterCreateSettings, config.Exporter) (component.LogsExporter, error) {
	return &ExampleExporter{}, nil
}

// ExampleExporter stores consumed traces and metrics for testing purposes.
type ExampleExporter struct {
	Traces  []ptrace.Traces
	Metrics []pmetric.Metrics
	Logs    []plog.Logs
	Started bool
	Stopped bool
}

// Start tells the exporter to start. The exporter may prepare for exporting
// by connecting to the endpoint. Host parameter can be used for communicating
// with the host after Start() has already returned.
func (exp *ExampleExporter) Start(_ context.Context, _ component.Host) error {
	exp.Started = true
	return nil
}

// ConsumeTraces receives ptrace.Traces for processing by the consumer.Traces.
func (exp *ExampleExporter) ConsumeTraces(_ context.Context, td ptrace.Traces) error {
	exp.Traces = append(exp.Traces, td)
	return nil
}

func (exp *ExampleExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeMetrics receives pmetric.Metrics for processing by the Metrics.
func (exp *ExampleExporter) ConsumeMetrics(_ context.Context, md pmetric.Metrics) error {
	exp.Metrics = append(exp.Metrics, md)
	return nil
}

func (exp *ExampleExporter) ConsumeLogs(_ context.Context, ld plog.Logs) error {
	exp.Logs = append(exp.Logs, ld)
	return nil
}

// Shutdown is invoked during shutdown.
func (exp *ExampleExporter) Shutdown(context.Context) error {
	exp.Stopped = true
	return nil
}
