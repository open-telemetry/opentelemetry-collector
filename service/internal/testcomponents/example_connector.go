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
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const connType = "exampleconnector"

// ExampleConnectorFactory is factory for ExampleConnector.
var ExampleConnectorFactory = connector.NewFactory(
	connType,
	createExampleConnectorDefaultConfig,

	connector.WithTracesToTraces(createExampleTracesToTraces, component.StabilityLevelDevelopment),
	connector.WithTracesToMetrics(createExampleTracesToMetrics, component.StabilityLevelDevelopment),
	connector.WithTracesToLogs(createExampleTracesToLogs, component.StabilityLevelDevelopment),

	connector.WithMetricsToTraces(createExampleMetricsToTraces, component.StabilityLevelDevelopment),
	connector.WithMetricsToMetrics(createExampleMetricsToMetrics, component.StabilityLevelDevelopment),
	connector.WithMetricsToLogs(createExampleMetricsToLogs, component.StabilityLevelDevelopment),

	connector.WithLogsToTraces(createExampleLogsToTraces, component.StabilityLevelDevelopment),
	connector.WithLogsToMetrics(createExampleLogsToMetrics, component.StabilityLevelDevelopment),
	connector.WithLogsToLogs(createExampleLogsToLogs, component.StabilityLevelDevelopment),
)

func createExampleConnectorDefaultConfig() component.Config {
	return &struct{}{}
}

func createExampleTracesToTraces(_ context.Context, _ connector.CreateSettings, _ component.Config, traces consumer.Traces) (connector.Traces, error) {
	return &ExampleConnector{
		ConsumeTracesFunc: traces.ConsumeTraces,
	}, nil
}

func createExampleTracesToMetrics(_ context.Context, _ connector.CreateSettings, _ component.Config, metrics consumer.Metrics) (connector.Traces, error) {
	return &ExampleConnector{
		ConsumeTracesFunc: func(ctx context.Context, td ptrace.Traces) error {
			return metrics.ConsumeMetrics(ctx, testdata.GenerateMetrics(td.SpanCount()))
		},
	}, nil
}

func createExampleTracesToLogs(_ context.Context, _ connector.CreateSettings, _ component.Config, logs consumer.Logs) (connector.Traces, error) {
	return &ExampleConnector{
		ConsumeTracesFunc: func(ctx context.Context, td ptrace.Traces) error {
			return logs.ConsumeLogs(ctx, testdata.GenerateLogs(td.SpanCount()))
		},
	}, nil
}

func createExampleMetricsToTraces(_ context.Context, _ connector.CreateSettings, _ component.Config, traces consumer.Traces) (connector.Metrics, error) {
	return &ExampleConnector{
		ConsumeMetricsFunc: func(ctx context.Context, md pmetric.Metrics) error {
			return traces.ConsumeTraces(ctx, testdata.GenerateTraces(md.MetricCount()))
		},
	}, nil
}

func createExampleMetricsToMetrics(_ context.Context, _ connector.CreateSettings, _ component.Config, metrics consumer.Metrics) (connector.Metrics, error) {
	return &ExampleConnector{
		ConsumeMetricsFunc: metrics.ConsumeMetrics,
	}, nil
}

func createExampleMetricsToLogs(_ context.Context, _ connector.CreateSettings, _ component.Config, logs consumer.Logs) (connector.Metrics, error) {
	return &ExampleConnector{
		ConsumeMetricsFunc: func(ctx context.Context, md pmetric.Metrics) error {
			return logs.ConsumeLogs(ctx, testdata.GenerateLogs(md.MetricCount()))
		},
	}, nil
}

func createExampleLogsToTraces(_ context.Context, _ connector.CreateSettings, _ component.Config, traces consumer.Traces) (connector.Logs, error) {
	return &ExampleConnector{
		ConsumeLogsFunc: func(ctx context.Context, ld plog.Logs) error {
			return traces.ConsumeTraces(ctx, testdata.GenerateTraces(ld.LogRecordCount()))
		},
	}, nil
}

func createExampleLogsToMetrics(_ context.Context, _ connector.CreateSettings, _ component.Config, metrics consumer.Metrics) (connector.Logs, error) {
	return &ExampleConnector{
		ConsumeLogsFunc: func(ctx context.Context, ld plog.Logs) error {
			return metrics.ConsumeMetrics(ctx, testdata.GenerateMetrics(ld.LogRecordCount()))
		},
	}, nil
}

func createExampleLogsToLogs(_ context.Context, _ connector.CreateSettings, _ component.Config, logs consumer.Logs) (connector.Logs, error) {
	return &ExampleConnector{
		ConsumeLogsFunc: logs.ConsumeLogs,
	}, nil
}

type ExampleConnector struct {
	componentState
	consumer.ConsumeTracesFunc
	consumer.ConsumeMetricsFunc
	consumer.ConsumeLogsFunc
}

func (c *ExampleConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}
