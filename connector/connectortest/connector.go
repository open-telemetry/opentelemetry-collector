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

var nopType = component.MustNewType("nop")

// NewNopCreateSettings returns a new nop settings for Create* functions.
//
// Deprecated: [v0.103.0] Use connectortest.NewNopSettings instead.
func NewNopCreateSettings() component.Settings {
	return componenttest.NewNopSettings()
}

type nopConfig struct{}

// NewNopFactory returns a connector.Factory that constructs nop processors.
func NewNopFactory() connector.Factory {
	return connector.NewFactory(
		nopType,
		func() component.Config {
			return &nopConfig{}
		},
		connector.WithTracesToTraces(createTracesToTracesConnector, component.StabilityLevelDevelopment),
		connector.WithTracesToMetrics(createTracesToMetricsConnector, component.StabilityLevelDevelopment),
		connector.WithTracesToLogs(createTracesToLogsConnector, component.StabilityLevelDevelopment),
		connector.WithMetricsToTraces(createMetricsToTracesConnector, component.StabilityLevelDevelopment),
		connector.WithMetricsToMetrics(createMetricsToMetricsConnector, component.StabilityLevelDevelopment),
		connector.WithMetricsToLogs(createMetricsToLogsConnector, component.StabilityLevelDevelopment),
		connector.WithLogsToTraces(createLogsToTracesConnector, component.StabilityLevelDevelopment),
		connector.WithLogsToMetrics(createLogsToMetricsConnector, component.StabilityLevelDevelopment),
		connector.WithLogsToLogs(createLogsToLogsConnector, component.StabilityLevelDevelopment),
	)
}

func createTracesToTracesConnector(context.Context, component.Settings, component.Config, consumer.Traces) (connector.Traces, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createTracesToMetricsConnector(context.Context, component.Settings, component.Config, consumer.Metrics) (connector.Traces, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createTracesToLogsConnector(context.Context, component.Settings, component.Config, consumer.Logs) (connector.Traces, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createMetricsToTracesConnector(context.Context, component.Settings, component.Config, consumer.Traces) (connector.Metrics, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createMetricsToMetricsConnector(context.Context, component.Settings, component.Config, consumer.Metrics) (connector.Metrics, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createMetricsToLogsConnector(context.Context, component.Settings, component.Config, consumer.Logs) (connector.Metrics, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createLogsToTracesConnector(context.Context, component.Settings, component.Config, consumer.Traces) (connector.Logs, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createLogsToMetricsConnector(context.Context, component.Settings, component.Config, consumer.Metrics) (connector.Logs, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createLogsToLogsConnector(context.Context, component.Settings, component.Config, consumer.Logs) (connector.Logs, error) {
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
	connID := component.NewIDWithName(nopType, "conn")
	return connector.NewBuilder(
		map[component.ID]component.Config{connID: nopFactory.CreateDefaultConfig()},
		map[component.Type]connector.Factory{nopType: nopFactory})
}
