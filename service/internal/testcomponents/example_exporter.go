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
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	typeStr   = "exampleexporter"
	stability = component.StabilityLevelDevelopment
)

// ExampleExporterFactory is factory for ExampleExporter.
var ExampleExporterFactory = exporter.NewFactory(
	typeStr,
	createExporterDefaultConfig,
	exporter.WithTraces(createTracesExporter, stability),
	exporter.WithMetrics(createMetricsExporter, stability),
	exporter.WithLogs(createLogsExporter, stability),
)

func createExporterDefaultConfig() component.Config {
	return &struct{}{}
}

func createTracesExporter(context.Context, exporter.CreateSettings, component.Config) (exporter.Traces, error) {
	return &ExampleExporter{}, nil
}

func createMetricsExporter(context.Context, exporter.CreateSettings, component.Config) (exporter.Metrics, error) {
	return &ExampleExporter{}, nil
}

func createLogsExporter(context.Context, exporter.CreateSettings, component.Config) (exporter.Logs, error) {
	return &ExampleExporter{}, nil
}

// ExampleExporter stores consumed traces and metrics for testing purposes.
type ExampleExporter struct {
	componentState
	Traces  []ptrace.Traces
	Metrics []pmetric.Metrics
	Logs    []plog.Logs
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

func (exp *ExampleExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}
