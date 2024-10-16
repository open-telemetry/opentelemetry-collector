// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connectortest // import "go.opentelemetry.io/collector/connector/connectortest"

import (
	"context"

	"github.com/google/uuid"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/connectorprofiles"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

var nopType = component.MustNewType("nop")

// NewNopSettings returns a new nop settings for Create* functions.
func NewNopSettings() connector.Settings {
	return connector.Settings{
		ID:                component.NewIDWithName(nopType, uuid.NewString()),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

type nopConfig struct{}

// NewNopFactory returns a connector.Factory that constructs nop processors.
func NewNopFactory() connector.Factory {
	return connectorprofiles.NewFactory(
		nopType,
		func() component.Config {
			return &nopConfig{}
		},
		connectorprofiles.WithTracesToTraces(createTracesToTracesConnector, component.StabilityLevelDevelopment),
		connectorprofiles.WithTracesToMetrics(createTracesToMetricsConnector, component.StabilityLevelDevelopment),
		connectorprofiles.WithTracesToLogs(createTracesToLogsConnector, component.StabilityLevelDevelopment),
		connectorprofiles.WithTracesToProfiles(createTracesToProfilesConnector, component.StabilityLevelAlpha),
		connectorprofiles.WithMetricsToTraces(createMetricsToTracesConnector, component.StabilityLevelDevelopment),
		connectorprofiles.WithMetricsToMetrics(createMetricsToMetricsConnector, component.StabilityLevelDevelopment),
		connectorprofiles.WithMetricsToLogs(createMetricsToLogsConnector, component.StabilityLevelDevelopment),
		connectorprofiles.WithMetricsToProfiles(createMetricsToProfilesConnector, component.StabilityLevelAlpha),
		connectorprofiles.WithLogsToTraces(createLogsToTracesConnector, component.StabilityLevelDevelopment),
		connectorprofiles.WithLogsToMetrics(createLogsToMetricsConnector, component.StabilityLevelDevelopment),
		connectorprofiles.WithLogsToLogs(createLogsToLogsConnector, component.StabilityLevelDevelopment),
		connectorprofiles.WithLogsToProfiles(createLogsToProfilesConnector, component.StabilityLevelAlpha),
		connectorprofiles.WithProfilesToTraces(createProfilesToTracesConnector, component.StabilityLevelAlpha),
		connectorprofiles.WithProfilesToMetrics(createProfilesToMetricsConnector, component.StabilityLevelAlpha),
		connectorprofiles.WithProfilesToLogs(createProfilesToLogsConnector, component.StabilityLevelAlpha),
		connectorprofiles.WithProfilesToProfiles(createProfilesToProfilesConnector, component.StabilityLevelAlpha),
	)
}

func createTracesToTracesConnector(context.Context, connector.Settings, component.Config, consumer.Traces) (connector.Traces, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createTracesToMetricsConnector(context.Context, connector.Settings, component.Config, consumer.Metrics) (connector.Traces, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createTracesToLogsConnector(context.Context, connector.Settings, component.Config, consumer.Logs) (connector.Traces, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createTracesToProfilesConnector(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (connector.Traces, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createMetricsToTracesConnector(context.Context, connector.Settings, component.Config, consumer.Traces) (connector.Metrics, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createMetricsToMetricsConnector(context.Context, connector.Settings, component.Config, consumer.Metrics) (connector.Metrics, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createMetricsToLogsConnector(context.Context, connector.Settings, component.Config, consumer.Logs) (connector.Metrics, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}
func createMetricsToProfilesConnector(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (connector.Metrics, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createLogsToTracesConnector(context.Context, connector.Settings, component.Config, consumer.Traces) (connector.Logs, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createLogsToMetricsConnector(context.Context, connector.Settings, component.Config, consumer.Metrics) (connector.Logs, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createLogsToLogsConnector(context.Context, connector.Settings, component.Config, consumer.Logs) (connector.Logs, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}
func createLogsToProfilesConnector(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (connector.Logs, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createProfilesToTracesConnector(context.Context, connector.Settings, component.Config, consumer.Traces) (connectorprofiles.Profiles, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createProfilesToMetricsConnector(context.Context, connector.Settings, component.Config, consumer.Metrics) (connectorprofiles.Profiles, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createProfilesToLogsConnector(context.Context, connector.Settings, component.Config, consumer.Logs) (connectorprofiles.Profiles, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}
func createProfilesToProfilesConnector(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (connectorprofiles.Profiles, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

// nopConnector stores consumed traces and metrics for testing purposes.
type nopConnector struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}
