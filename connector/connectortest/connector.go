// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connectortest // import "go.opentelemetry.io/collector/connector/connectortest"

import (
	"context"

	"github.com/google/uuid"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/xconnector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/xconsumer"
)

var NopType = component.MustNewType("nop")

// NewNopSettings returns a new nop settings for Create* functions with the given type.
func NewNopSettings(typ component.Type) connector.Settings {
	return connector.Settings{
		ID:                component.NewIDWithName(typ, uuid.NewString()),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

type nopConfig struct{}

// NewNopFactory returns a connector.Factory that constructs nop processors.
func NewNopFactory() connector.Factory {
	return xconnector.NewFactory(
		NopType,
		func() component.Config {
			return &nopConfig{}
		},
		xconnector.WithTracesToTraces(createTracesToTracesConnector, component.StabilityLevelDevelopment),
		xconnector.WithTracesToMetrics(createTracesToMetricsConnector, component.StabilityLevelDevelopment),
		xconnector.WithTracesToLogs(createTracesToLogsConnector, component.StabilityLevelDevelopment),
		xconnector.WithTracesToProfiles(createTracesToProfilesConnector, component.StabilityLevelAlpha),
		xconnector.WithMetricsToTraces(createMetricsToTracesConnector, component.StabilityLevelDevelopment),
		xconnector.WithMetricsToMetrics(createMetricsToMetricsConnector, component.StabilityLevelDevelopment),
		xconnector.WithMetricsToLogs(createMetricsToLogsConnector, component.StabilityLevelDevelopment),
		xconnector.WithMetricsToProfiles(createMetricsToProfilesConnector, component.StabilityLevelAlpha),
		xconnector.WithLogsToTraces(createLogsToTracesConnector, component.StabilityLevelDevelopment),
		xconnector.WithLogsToMetrics(createLogsToMetricsConnector, component.StabilityLevelDevelopment),
		xconnector.WithLogsToLogs(createLogsToLogsConnector, component.StabilityLevelDevelopment),
		xconnector.WithLogsToProfiles(createLogsToProfilesConnector, component.StabilityLevelAlpha),
		xconnector.WithProfilesToTraces(createProfilesToTracesConnector, component.StabilityLevelAlpha),
		xconnector.WithProfilesToMetrics(createProfilesToMetricsConnector, component.StabilityLevelAlpha),
		xconnector.WithProfilesToLogs(createProfilesToLogsConnector, component.StabilityLevelAlpha),
		xconnector.WithProfilesToProfiles(createProfilesToProfilesConnector, component.StabilityLevelAlpha),
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

func createTracesToProfilesConnector(context.Context, connector.Settings, component.Config, xconsumer.Profiles) (connector.Traces, error) {
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

func createMetricsToProfilesConnector(context.Context, connector.Settings, component.Config, xconsumer.Profiles) (connector.Metrics, error) {
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

func createLogsToProfilesConnector(context.Context, connector.Settings, component.Config, xconsumer.Profiles) (connector.Logs, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createProfilesToTracesConnector(context.Context, connector.Settings, component.Config, consumer.Traces) (xconnector.Profiles, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createProfilesToMetricsConnector(context.Context, connector.Settings, component.Config, consumer.Metrics) (xconnector.Profiles, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createProfilesToLogsConnector(context.Context, connector.Settings, component.Config, consumer.Logs) (xconnector.Profiles, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createProfilesToProfilesConnector(context.Context, connector.Settings, component.Config, xconsumer.Profiles) (xconnector.Profiles, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

// nopConnector stores consumed traces and metrics for testing purposes.
type nopConnector struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}
