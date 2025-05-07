// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testcomponents // import "go.opentelemetry.io/collector/service/internal/testcomponents"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/xconnector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pipeline"
)

var routerType = component.MustNewType("examplerouter")

// ExampleRouterFactory is factory for ExampleRouter.
var ExampleRouterFactory = xconnector.NewFactory(
	routerType,
	createExampleRouterDefaultConfig,
	xconnector.WithTracesToTraces(createExampleTracesRouter, component.StabilityLevelDevelopment),
	xconnector.WithMetricsToMetrics(createExampleMetricsRouter, component.StabilityLevelDevelopment),
	xconnector.WithLogsToLogs(createExampleLogsRouter, component.StabilityLevelDevelopment),
	xconnector.WithProfilesToProfiles(createExampleProfilesRouter, component.StabilityLevelDevelopment),
)

type LeftRightConfig struct {
	Left  pipeline.ID `mapstructure:"left"`
	Right pipeline.ID `mapstructure:"right"`
}

type ExampleRouterConfig struct {
	Traces   *LeftRightConfig `mapstructure:"traces"`
	Metrics  *LeftRightConfig `mapstructure:"metrics"`
	Logs     *LeftRightConfig `mapstructure:"logs"`
	Profiles *LeftRightConfig `mapstructure:"profiles"`
}

func createExampleRouterDefaultConfig() component.Config {
	return &ExampleRouterConfig{}
}

func createExampleTracesRouter(_ context.Context, _ connector.Settings, cfg component.Config, traces consumer.Traces) (connector.Traces, error) {
	c := cfg.(ExampleRouterConfig)
	r := traces.(connector.TracesRouterAndConsumer)
	left, _ := r.Consumer(c.Traces.Left)
	right, _ := r.Consumer(c.Traces.Right)
	return &ExampleRouter{
		tracesRight: right,
		tracesLeft:  left,
	}, nil
}

func createExampleMetricsRouter(_ context.Context, _ connector.Settings, cfg component.Config, metrics consumer.Metrics) (connector.Metrics, error) {
	c := cfg.(ExampleRouterConfig)
	r := metrics.(connector.MetricsRouterAndConsumer)
	left, _ := r.Consumer(c.Metrics.Left)
	right, _ := r.Consumer(c.Metrics.Right)
	return &ExampleRouter{
		metricsRight: right,
		metricsLeft:  left,
	}, nil
}

func createExampleLogsRouter(_ context.Context, _ connector.Settings, cfg component.Config, logs consumer.Logs) (connector.Logs, error) {
	c := cfg.(ExampleRouterConfig)
	r := logs.(connector.LogsRouterAndConsumer)
	left, _ := r.Consumer(c.Logs.Left)
	right, _ := r.Consumer(c.Logs.Right)
	return &ExampleRouter{
		logsRight: right,
		logsLeft:  left,
	}, nil
}

func createExampleProfilesRouter(_ context.Context, _ connector.Settings, cfg component.Config, profiles xconsumer.Profiles) (xconnector.Profiles, error) {
	c := cfg.(ExampleRouterConfig)
	r := profiles.(xconnector.ProfilesRouterAndConsumer)
	left, _ := r.Consumer(c.Profiles.Left)
	right, _ := r.Consumer(c.Profiles.Right)
	return &ExampleRouter{
		profilesRight: right,
		profilesLeft:  left,
	}, nil
}

type ExampleRouter struct {
	componentState

	tracesRight consumer.Traces
	tracesLeft  consumer.Traces
	tracesNum   int

	metricsRight consumer.Metrics
	metricsLeft  consumer.Metrics
	metricsNum   int

	logsRight consumer.Logs
	logsLeft  consumer.Logs
	logsNum   int

	profilesRight xconsumer.Profiles
	profilesLeft  xconsumer.Profiles
	profilesNum   int
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

func (r *ExampleRouter) ConsumeProfiles(ctx context.Context, td pprofile.Profiles) error {
	r.profilesNum++
	if r.profilesNum%2 == 0 {
		return r.profilesLeft.ConsumeProfiles(ctx, td)
	}
	return r.profilesRight.ConsumeProfiles(ctx, td)
}

func (r *ExampleRouter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}
