// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connectortest // import "go.opentelemetry.io/collector/connector/connectortest"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

const typeStr = "nop"

// NewNopCreateSettings returns a new nop settings for Create* functions.
func NewNopCreateSettings() connector.CreateSettings {
	return connector.CreateSettings{
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

type nopConfig struct{}

// NewNopFactory returns a connector.Factory that constructs nop processors.
func NewNopFactory(opts ...connector.FactoryOption) connector.Factory {
	if len(opts) == 0 {
		opts = []connector.FactoryOption{
			connector.WithTracesToTraces(CreateTracesToTracesConnector, component.StabilityLevelDevelopment),
			connector.WithTracesToMetrics(CreateTracesToMetricsConnector, component.StabilityLevelDevelopment),
			connector.WithTracesToLogs(CreateTracesToLogsConnector, component.StabilityLevelDevelopment),
			connector.WithMetricsToTraces(CreateMetricsToTracesConnector, component.StabilityLevelDevelopment),
			connector.WithMetricsToMetrics(CreateMetricsToMetricsConnector, component.StabilityLevelDevelopment),
			connector.WithMetricsToLogs(CreateMetricsToLogsConnector, component.StabilityLevelDevelopment),
			connector.WithLogsToTraces(CreateLogsToTracesConnector, component.StabilityLevelDevelopment),
			connector.WithLogsToMetrics(CreateLogsToMetricsConnector, component.StabilityLevelDevelopment),
			connector.WithLogsToLogs(CreateLogsToLogsConnector, component.StabilityLevelDevelopment),
		}
	}
	return connector.NewFactory(
		"nop",
		func() component.Config {
			return &nopConfig{}
		},
		opts...,
	)
}

func CreateTracesToTracesConnector(context.Context, connector.CreateSettings, component.Config, consumer.Traces) (connector.Traces, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func CreateTracesToMetricsConnector(context.Context, connector.CreateSettings, component.Config, consumer.Metrics) (connector.Traces, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func CreateTracesToLogsConnector(context.Context, connector.CreateSettings, component.Config, consumer.Logs) (connector.Traces, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func CreateMetricsToTracesConnector(context.Context, connector.CreateSettings, component.Config, consumer.Traces) (connector.Metrics, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func CreateMetricsToMetricsConnector(context.Context, connector.CreateSettings, component.Config, consumer.Metrics) (connector.Metrics, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func CreateMetricsToLogsConnector(context.Context, connector.CreateSettings, component.Config, consumer.Logs) (connector.Metrics, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func CreateLogsToTracesConnector(context.Context, connector.CreateSettings, component.Config, consumer.Traces) (connector.Logs, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func CreateLogsToMetricsConnector(context.Context, connector.CreateSettings, component.Config, consumer.Metrics) (connector.Logs, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func CreateLogsToLogsConnector(context.Context, connector.CreateSettings, component.Config, consumer.Logs) (connector.Logs, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

// nopConnector stores consumed traces and metrics for testing purposes.
type nopConnector struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}

// NewNopBuilder returns a connector.Builder that constructs nop receivers.
func NewNopBuilder() *connector.Builder {
	nopFactory := NewNopFactory()
	// Use a different ID than receivertest and exportertest to avoid ambiguous
	// configuration scenarios. Ambiguous IDs are detected in the 'otelcol' package,
	// but lower level packages such as 'service' assume that IDs are disambiguated.
	connID := component.NewIDWithName(typeStr, "conn")
	return connector.NewBuilder(
		map[component.ID]component.Config{connID: nopFactory.CreateDefaultConfig()},
		map[component.Type]connector.Factory{typeStr: nopFactory})
}
