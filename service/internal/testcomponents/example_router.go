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
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const routerType = "examplerouter"

// ExampleRouterFactory is factory for ExampleRouter.
var ExampleRouterFactory = connector.NewFactory(
	routerType,
	createExampleRouterDefaultConfig,
	connector.WithTracesToTraces(createExampleTracesRouter, component.StabilityLevelDevelopment),
	connector.WithMetricsToMetrics(createExampleMetricsRouter, component.StabilityLevelDevelopment),
	connector.WithLogsToLogs(createExampleLogsRouter, component.StabilityLevelDevelopment),
)

type LeftRightConfig struct {
	Left  component.ID `mapstructure:"left"`
	Right component.ID `mapstructure:"right"`
}

type ExampleRouterConfig struct {
	Traces  *LeftRightConfig `mapstructure:"traces"`
	Metrics *LeftRightConfig `mapstructure:"metrics"`
	Logs    *LeftRightConfig `mapstructure:"logs"`
}

func createExampleRouterDefaultConfig() component.Config {
	return &ExampleRouterConfig{}
}

func createExampleTracesRouter(_ context.Context, _ connector.CreateSettings, cfg component.Config, traces consumer.Traces) (connector.Traces, error) {
	c := cfg.(ExampleRouterConfig)
	cons := traces.(connector.TracesConsumer).Consumers()
	return &ExampleRouter{
		traces:      traces.(connector.TracesConsumer),
		tracesRight: cons[c.Traces.Right],
		tracesLeft:  cons[c.Traces.Left],
	}, nil
}

func createExampleMetricsRouter(_ context.Context, _ connector.CreateSettings, cfg component.Config, metrics consumer.Metrics) (connector.Metrics, error) {
	c := cfg.(ExampleRouterConfig)
	cons := metrics.(connector.MetricsConsumer).Consumers()
	return &ExampleRouter{
		metrics:      metrics.(connector.MetricsConsumer),
		metricsRight: cons[c.Metrics.Right],
		metricsLeft:  cons[c.Metrics.Left],
	}, nil
}

func createExampleLogsRouter(_ context.Context, _ connector.CreateSettings, cfg component.Config, logs consumer.Logs) (connector.Logs, error) {
	c := cfg.(ExampleRouterConfig)
	cons := logs.(connector.LogsConsumer).Consumers()
	return &ExampleRouter{
		logs:      logs.(connector.LogsConsumer),
		logsRight: cons[c.Logs.Right],
		logsLeft:  cons[c.Logs.Left],
	}, nil
}

type ExampleRouter struct {
	componentState

	traces      connector.TracesConsumer
	tracesRight consumer.Traces
	tracesLeft  consumer.Traces
	tracesNum   int

	metrics      connector.MetricsConsumer
	metricsRight consumer.Metrics
	metricsLeft  consumer.Metrics
	metricsNum   int

	logs      connector.LogsConsumer
	logsRight consumer.Logs
	logsLeft  consumer.Logs
	logsNum   int
}

func (r *ExampleRouter) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	r.tracesNum++
	if r.tracesNum%2 == 0 {
		return r.tracesLeft.ConsumeTraces(ctx, td)
	}
	return r.tracesRight.ConsumeTraces(ctx, td)
}

func (r *ExampleRouter) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	r.metricsNum++
	if r.metricsNum%2 == 0 {
		return r.metricsLeft.ConsumeMetrics(ctx, md)
	}
	return r.metricsRight.ConsumeMetrics(ctx, md)
}

func (r *ExampleRouter) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	r.logsNum++
	if r.logsNum%2 == 0 {
		return r.logsLeft.ConsumeLogs(ctx, ld)
	}
	return r.logsRight.ConsumeLogs(ctx, ld)
}

func (r *ExampleRouter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}
