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
	return &ExampleRouter{
		traces:      traces.(connector.TracesRouter),
		tracesRight: c.Traces.Right,
		tracesLeft:  c.Traces.Left,
	}, nil
}

func createExampleMetricsRouter(_ context.Context, _ connector.CreateSettings, cfg component.Config, metrics consumer.Metrics) (connector.Metrics, error) {
	c := cfg.(ExampleRouterConfig)
	return &ExampleRouter{
		metrics:      metrics.(connector.MetricsRouter),
		metricsRight: c.Metrics.Right,
		metricsLeft:  c.Metrics.Left,
	}, nil
}

func createExampleLogsRouter(_ context.Context, _ connector.CreateSettings, cfg component.Config, logs consumer.Logs) (connector.Logs, error) {
	c := cfg.(ExampleRouterConfig)
	return &ExampleRouter{
		logs:      logs.(connector.LogsRouter),
		logsRight: c.Logs.Right,
		logsLeft:  c.Logs.Left,
	}, nil
}

type ExampleRouter struct {
	componentState

	traces      connector.TracesRouter
	tracesRight component.ID
	tracesLeft  component.ID
	tracesNum   int

	metrics      connector.MetricsRouter
	metricsRight component.ID
	metricsLeft  component.ID
	metricsNum   int

	logs      connector.LogsRouter
	logsRight component.ID
	logsLeft  component.ID
	logsNum   int
}

func (r *ExampleRouter) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	r.tracesNum++
	if r.tracesNum%2 == 0 {
		return r.traces.RouteTraces(ctx, td, r.tracesLeft)
	}
	return r.traces.RouteTraces(ctx, td, r.tracesRight)
}

func (r *ExampleRouter) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	r.metricsNum++
	if r.metricsNum%2 == 0 {
		return r.metrics.RouteMetrics(ctx, md, r.metricsLeft)
	}
	return r.metrics.RouteMetrics(ctx, md, r.metricsRight)
}

func (r *ExampleRouter) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	r.logsNum++
	if r.logsNum%2 == 0 {
		return r.logs.RouteLogs(ctx, ld, r.logsLeft)
	}
	return r.logs.RouteLogs(ctx, ld, r.logsRight)
}

func (r *ExampleRouter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}
