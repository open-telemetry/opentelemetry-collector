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
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const connType = "exampleconnector"

// ExampleConnectorConfig config for ExampleConnector.
type ExampleConnectorConfig struct {
	config.ConnectorSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
}

// ExampleConnectorFactory is factory for ExampleConnector.
var ExampleConnectorFactory = component.NewConnectorFactory(
	connType,
	createExampleConnectorDefaultConfig,

	component.WithTracesToTracesConnector(createExampleTracesToTracesConnector, component.StabilityLevelDevelopment),
	component.WithTracesToMetricsConnector(createTranslateTracesToMetricsConnector, component.StabilityLevelDevelopment),
	component.WithTracesToLogsConnector(createTranslateTracesToLogsConnector, component.StabilityLevelDevelopment),

	component.WithMetricsToTracesConnector(createTranslateMetricsToTracesConnector, component.StabilityLevelDevelopment),
	component.WithMetricsToMetricsConnector(createExampleMetricsToMetricsConnector, component.StabilityLevelDevelopment),
	component.WithMetricsToLogsConnector(createTranslateMetricsToLogsConnector, component.StabilityLevelDevelopment),

	component.WithLogsToTracesConnector(createTranslateLogsToTracesConnector, component.StabilityLevelDevelopment),
	component.WithLogsToMetricsConnector(createTranslateLogsToMetricsConnector, component.StabilityLevelDevelopment),
	component.WithLogsToLogsConnector(createExampleLogsToLogsConnector, component.StabilityLevelDevelopment),
)

func createExampleConnectorDefaultConfig() component.Config {
	return &ExampleConnectorConfig{
		ConnectorSettings: config.NewConnectorSettings(component.NewID(typeStr)),
	}
}

func createExampleTracesToTracesConnector(_ context.Context, _ component.ConnectorCreateSettings, _ component.Config, traces consumer.Traces) (component.TracesToTracesConnector, error) {
	return &ExampleConnector{Traces: traces}, nil
}
func createTranslateTracesToMetricsConnector(_ context.Context, _ component.ConnectorCreateSettings, _ component.Config, metrics consumer.Metrics) (component.TracesToMetricsConnector, error) {
	return &ExampleConnector{Metrics: metrics}, nil
}
func createTranslateTracesToLogsConnector(_ context.Context, _ component.ConnectorCreateSettings, _ component.Config, logs consumer.Logs) (component.TracesToLogsConnector, error) {
	return &ExampleConnector{Logs: logs}, nil
}

func createTranslateMetricsToTracesConnector(_ context.Context, _ component.ConnectorCreateSettings, _ component.Config, traces consumer.Traces) (component.MetricsToTracesConnector, error) {
	return &ExampleConnector{Traces: traces}, nil
}
func createExampleMetricsToMetricsConnector(_ context.Context, _ component.ConnectorCreateSettings, _ component.Config, metrics consumer.Metrics) (component.MetricsToMetricsConnector, error) {
	return &ExampleConnector{Metrics: metrics}, nil
}
func createTranslateMetricsToLogsConnector(_ context.Context, _ component.ConnectorCreateSettings, _ component.Config, logs consumer.Logs) (component.MetricsToLogsConnector, error) {
	return &ExampleConnector{Logs: logs}, nil
}

func createTranslateLogsToTracesConnector(_ context.Context, _ component.ConnectorCreateSettings, _ component.Config, traces consumer.Traces) (component.LogsToTracesConnector, error) {
	return &ExampleConnector{Traces: traces}, nil
}
func createTranslateLogsToMetricsConnector(_ context.Context, _ component.ConnectorCreateSettings, _ component.Config, metrics consumer.Metrics) (component.LogsToMetricsConnector, error) {
	return &ExampleConnector{Metrics: metrics}, nil
}
func createExampleLogsToLogsConnector(_ context.Context, _ component.ConnectorCreateSettings, _ component.Config, logs consumer.Logs) (component.LogsToLogsConnector, error) {
	return &ExampleConnector{Logs: logs}, nil
}

var _ StatefulComponent = &ExampleConnector{}

type ExampleConnector struct {
	Traces  consumer.Traces
	Metrics consumer.Metrics
	Logs    consumer.Logs
	componentState
}

// Start tells the Connector to start.
func (c *ExampleConnector) Start(_ context.Context, _ component.Host) error {
	c.started = true
	return nil
}

// ConsumeTraces receives ptrace.Traces for processing by the consumer.Traces.
func (c *ExampleConnector) ConsumeTracesToTraces(ctx context.Context, td ptrace.Traces) error {
	return c.Traces.ConsumeTraces(ctx, td)
}

// ConsumeTracesToMetrics receives ptrace.Traces and emits pmetric.Metrics to consumer.Metrics.
func (c *ExampleConnector) ConsumeTracesToMetrics(ctx context.Context, td ptrace.Traces) error {
	return c.Metrics.ConsumeMetrics(ctx, testdata.GenerateMetrics(td.SpanCount()))
}

// ConsumeTracesToLogs receives ptrace.Traces and emits pmetric.Logs to consumer.Logs.
func (c *ExampleConnector) ConsumeTracesToLogs(ctx context.Context, td ptrace.Traces) error {
	return c.Logs.ConsumeLogs(ctx, testdata.GenerateLogs(td.SpanCount()))
}

// ConsumeMetricsToTraces receives pmetric.Metrics and emits pmetric.Traces to consumer.Traces.
func (c *ExampleConnector) ConsumeMetricsToTraces(ctx context.Context, md pmetric.Metrics) error {
	return c.Traces.ConsumeTraces(ctx, testdata.GenerateTraces(md.MetricCount()))
}

// ConsumeMetrics receives pmetric.Metrics for processing by the Metrics.
func (c *ExampleConnector) ConsumeMetricsToMetrics(ctx context.Context, md pmetric.Metrics) error {
	return c.Metrics.ConsumeMetrics(ctx, md)
}

// ConsumeMetricsToLogs receives pmetric.Metrics and emits pmetric.Logs to consumer.Logs.
func (c *ExampleConnector) ConsumeMetricsToLogs(ctx context.Context, md pmetric.Metrics) error {
	return c.Logs.ConsumeLogs(ctx, testdata.GenerateLogs(md.MetricCount()))
}

// ConsumeLogsToTraces receives plog.Logs and emits pmetric.Traces to consumer.Traces.
func (c *ExampleConnector) ConsumeLogsToTraces(ctx context.Context, ld plog.Logs) error {
	return c.Traces.ConsumeTraces(ctx, testdata.GenerateTraces(ld.LogRecordCount()))
}

// ConsumeLogsToMetrics receives plog.Logs and emits pmetric.Metrics to consumer.Metrics.
func (c *ExampleConnector) ConsumeLogsToMetrics(ctx context.Context, ld plog.Logs) error {
	return c.Metrics.ConsumeMetrics(ctx, testdata.GenerateMetrics(ld.LogRecordCount()))
}

// ConsumeLogs receives pmetric.Logs for processing by the Metrics.
func (c *ExampleConnector) ConsumeLogsToLogs(ctx context.Context, ld plog.Logs) error {
	return c.Logs.ConsumeLogs(ctx, ld)
}

// Shutdown is invoked during shutdown.
func (c *ExampleConnector) Shutdown(context.Context) error {
	c.stopped = true
	return nil
}

func (c *ExampleConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}
