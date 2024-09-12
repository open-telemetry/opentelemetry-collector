// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testcomponents // import "go.opentelemetry.io/collector/service/internal/testcomponents"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/connectorprofiles"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
)

var connType = component.MustNewType("exampleconnector")

// ExampleConnectorFactory is factory for ExampleConnector.
var ExampleConnectorFactory = connector.NewFactory(
	connType,
	createExampleConnectorDefaultConfig,

	connector.WithTracesToTraces(createExampleTracesToTraces, component.StabilityLevelDevelopment),
	connector.WithTracesToMetrics(createExampleTracesToMetrics, component.StabilityLevelDevelopment),
	connector.WithTracesToLogs(createExampleTracesToLogs, component.StabilityLevelDevelopment),
	connectorprofiles.WithTracesToProfiles(createExampleTracesToProfiles, component.StabilityLevelDevelopment),

	connector.WithMetricsToTraces(createExampleMetricsToTraces, component.StabilityLevelDevelopment),
	connector.WithMetricsToMetrics(createExampleMetricsToMetrics, component.StabilityLevelDevelopment),
	connector.WithMetricsToLogs(createExampleMetricsToLogs, component.StabilityLevelDevelopment),
	connectorprofiles.WithMetricsToProfiles(createExampleMetricsToProfiles, component.StabilityLevelDevelopment),

	connector.WithLogsToTraces(createExampleLogsToTraces, component.StabilityLevelDevelopment),
	connector.WithLogsToMetrics(createExampleLogsToMetrics, component.StabilityLevelDevelopment),
	connector.WithLogsToLogs(createExampleLogsToLogs, component.StabilityLevelDevelopment),
	connectorprofiles.WithLogsToProfiles(createExampleLogsToProfiles, component.StabilityLevelDevelopment),

	connectorprofiles.WithProfilesToTraces(createExampleProfilesToTraces, component.StabilityLevelDevelopment),
	connectorprofiles.WithProfilesToMetrics(createExampleProfilesToMetrics, component.StabilityLevelDevelopment),
	connectorprofiles.WithProfilesToLogs(createExampleProfilesToLogs, component.StabilityLevelDevelopment),
	connectorprofiles.WithProfilesToProfiles(createExampleProfilesToProfiles, component.StabilityLevelDevelopment),
)

var MockForwardConnectorFactory = connector.NewFactory(
	component.MustNewType("mockforward"),
	createExampleConnectorDefaultConfig,
	connector.WithTracesToTraces(createExampleTracesToTraces, component.StabilityLevelDevelopment),
	connector.WithMetricsToMetrics(createExampleMetricsToMetrics, component.StabilityLevelDevelopment),
	connector.WithLogsToLogs(createExampleLogsToLogs, component.StabilityLevelDevelopment),
	connectorprofiles.WithProfilesToProfiles(createExampleProfilesToProfiles, component.StabilityLevelDevelopment),
)

func createExampleConnectorDefaultConfig() component.Config {
	return &struct{}{}
}

func createExampleTracesToTraces(_ context.Context, set connector.Settings, _ component.Config, traces consumer.Traces) (connector.Traces, error) {
	return &ExampleConnector{
		ConsumeTracesFunc: traces.ConsumeTraces,
		mutatesData:       set.ID.Name() == "mutate",
	}, nil
}

func createExampleTracesToMetrics(_ context.Context, set connector.Settings, _ component.Config, metrics consumer.Metrics) (connector.Traces, error) {
	return &ExampleConnector{
		ConsumeTracesFunc: func(ctx context.Context, td ptrace.Traces) error {
			return metrics.ConsumeMetrics(ctx, testdata.GenerateMetrics(td.SpanCount()))
		},
		mutatesData: set.ID.Name() == "mutate",
	}, nil
}

func createExampleTracesToLogs(_ context.Context, set connector.Settings, _ component.Config, logs consumer.Logs) (connector.Traces, error) {
	return &ExampleConnector{
		ConsumeTracesFunc: func(ctx context.Context, td ptrace.Traces) error {
			return logs.ConsumeLogs(ctx, testdata.GenerateLogs(td.SpanCount()))
		},
		mutatesData: set.ID.Name() == "mutate",
	}, nil
}

func createExampleTracesToProfiles(_ context.Context, set connector.Settings, _ component.Config, profiles consumerprofiles.Profiles) (connector.Traces, error) {
	return &ExampleConnector{
		ConsumeTracesFunc: func(ctx context.Context, td ptrace.Traces) error {
			return profiles.ConsumeProfiles(ctx, testdata.GenerateProfiles(td.SpanCount()))
		},
		mutatesData: set.ID.Name() == "mutate",
	}, nil
}

func createExampleMetricsToTraces(_ context.Context, set connector.Settings, _ component.Config, traces consumer.Traces) (connector.Metrics, error) {
	return &ExampleConnector{
		ConsumeMetricsFunc: func(ctx context.Context, md pmetric.Metrics) error {
			return traces.ConsumeTraces(ctx, testdata.GenerateTraces(md.MetricCount()))
		},
		mutatesData: set.ID.Name() == "mutate",
	}, nil
}

func createExampleMetricsToMetrics(_ context.Context, set connector.Settings, _ component.Config, metrics consumer.Metrics) (connector.Metrics, error) {
	return &ExampleConnector{
		ConsumeMetricsFunc: metrics.ConsumeMetrics,
		mutatesData:        set.ID.Name() == "mutate",
	}, nil
}

func createExampleMetricsToLogs(_ context.Context, set connector.Settings, _ component.Config, logs consumer.Logs) (connector.Metrics, error) {
	return &ExampleConnector{
		ConsumeMetricsFunc: func(ctx context.Context, md pmetric.Metrics) error {
			return logs.ConsumeLogs(ctx, testdata.GenerateLogs(md.MetricCount()))
		},
		mutatesData: set.ID.Name() == "mutate",
	}, nil
}

func createExampleMetricsToProfiles(_ context.Context, set connector.Settings, _ component.Config, profiles consumerprofiles.Profiles) (connector.Metrics, error) {
	return &ExampleConnector{
		ConsumeMetricsFunc: func(ctx context.Context, md pmetric.Metrics) error {
			return profiles.ConsumeProfiles(ctx, testdata.GenerateProfiles(md.MetricCount()))
		},
		mutatesData: set.ID.Name() == "mutate",
	}, nil
}

func createExampleLogsToTraces(_ context.Context, set connector.Settings, _ component.Config, traces consumer.Traces) (connector.Logs, error) {
	return &ExampleConnector{
		ConsumeLogsFunc: func(ctx context.Context, ld plog.Logs) error {
			return traces.ConsumeTraces(ctx, testdata.GenerateTraces(ld.LogRecordCount()))
		},
		mutatesData: set.ID.Name() == "mutate",
	}, nil
}

func createExampleLogsToMetrics(_ context.Context, set connector.Settings, _ component.Config, metrics consumer.Metrics) (connector.Logs, error) {
	return &ExampleConnector{
		ConsumeLogsFunc: func(ctx context.Context, ld plog.Logs) error {
			return metrics.ConsumeMetrics(ctx, testdata.GenerateMetrics(ld.LogRecordCount()))
		},
		mutatesData: set.ID.Name() == "mutate",
	}, nil
}

func createExampleLogsToLogs(_ context.Context, set connector.Settings, _ component.Config, logs consumer.Logs) (connector.Logs, error) {
	return &ExampleConnector{
		ConsumeLogsFunc: logs.ConsumeLogs,
		mutatesData:     set.ID.Name() == "mutate",
	}, nil
}

func createExampleLogsToProfiles(_ context.Context, set connector.Settings, _ component.Config, profiles consumerprofiles.Profiles) (connector.Logs, error) {
	return &ExampleConnector{
		ConsumeLogsFunc: func(ctx context.Context, ld plog.Logs) error {
			return profiles.ConsumeProfiles(ctx, testdata.GenerateProfiles(ld.LogRecordCount()))
		},
		mutatesData: set.ID.Name() == "mutate",
	}, nil
}

func createExampleProfilesToTraces(_ context.Context, set connector.Settings, _ component.Config, traces consumer.Traces) (connectorprofiles.Profiles, error) {
	return &ExampleConnector{
		ConsumeProfilesFunc: func(ctx context.Context, _ pprofile.Profiles) error {
			return traces.ConsumeTraces(ctx, testdata.GenerateTraces(1))
		},
		mutatesData: set.ID.Name() == "mutate",
	}, nil
}

func createExampleProfilesToMetrics(_ context.Context, set connector.Settings, _ component.Config, metrics consumer.Metrics) (connectorprofiles.Profiles, error) {
	return &ExampleConnector{
		ConsumeProfilesFunc: func(ctx context.Context, _ pprofile.Profiles) error {
			return metrics.ConsumeMetrics(ctx, testdata.GenerateMetrics(1))
		},
		mutatesData: set.ID.Name() == "mutate",
	}, nil
}

func createExampleProfilesToLogs(_ context.Context, set connector.Settings, _ component.Config, logs consumer.Logs) (connectorprofiles.Profiles, error) {
	return &ExampleConnector{
		ConsumeProfilesFunc: func(ctx context.Context, _ pprofile.Profiles) error {
			return logs.ConsumeLogs(ctx, testdata.GenerateLogs(1))
		},
		mutatesData: set.ID.Name() == "mutate",
	}, nil
}
func createExampleProfilesToProfiles(_ context.Context, set connector.Settings, _ component.Config, profiles consumerprofiles.Profiles) (connectorprofiles.Profiles, error) {
	return &ExampleConnector{
		ConsumeProfilesFunc: profiles.ConsumeProfiles,
		mutatesData:         set.ID.Name() == "mutate",
	}, nil
}

type ExampleConnector struct {
	componentState
	consumer.ConsumeTracesFunc
	consumer.ConsumeMetricsFunc
	consumer.ConsumeLogsFunc
	consumerprofiles.ConsumeProfilesFunc
	mutatesData bool
}

func (c *ExampleConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: c.mutatesData}
}
