// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package forwardconnector // import "go.opentelemetry.io/collector/connector/forwardconnector"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/forwardconnector/internal/metadata"
	"go.opentelemetry.io/collector/connector/xconnector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
)

// NewFactory returns a connector.Factory.
func NewFactory() xconnector.Factory {
	return xconnector.NewFactory(
		metadata.Type,
		createDefaultConfig,
		xconnector.WithTracesToTraces(createTracesToTraces, metadata.TracesToTracesStability),
		xconnector.WithMetricsToMetrics(createMetricsToMetrics, metadata.MetricsToMetricsStability),
		xconnector.WithLogsToLogs(createLogsToLogs, metadata.LogsToLogsStability),
		xconnector.WithProfilesToProfiles(createProfilesToProfiles, metadata.ProfilesToProfilesStability),
	)
}

type Config struct{}

// createDefaultConfig creates the default configuration.
func createDefaultConfig() component.Config {
	return &Config{}
}

// createTracesToTraces creates a trace receiver based on provided config.
func createTracesToTraces(
	_ context.Context,
	_ connector.Settings,
	_ component.Config,
	nextConsumer consumer.Traces,
) (connector.Traces, error) {
	return &forward{Traces: nextConsumer}, nil
}

// createMetricsToMetrics creates a metrics receiver based on provided config.
func createMetricsToMetrics(
	_ context.Context,
	_ connector.Settings,
	_ component.Config,
	nextConsumer consumer.Metrics,
) (connector.Metrics, error) {
	return &forward{Metrics: nextConsumer}, nil
}

// createLogsToLogs creates a log receiver based on provided config.
func createLogsToLogs(
	_ context.Context,
	_ connector.Settings,
	_ component.Config,
	nextConsumer consumer.Logs,
) (connector.Logs, error) {
	return &forward{Logs: nextConsumer}, nil
}

// createProfilesToProfiles creates a profile receiver based on provided config.
func createProfilesToProfiles(
	_ context.Context,
	_ connector.Settings,
	_ component.Config,
	nextConsumer xconsumer.Profiles,
) (xconnector.Profiles, error) {
	return &forward{Profiles: nextConsumer}, nil
}

// forward is used to pass signals directly from one pipeline to another.
// This is useful when there is a need to replicate data and process it in more
// than one way. It can also be used to join pipelines together.
type forward struct {
	consumer.Traces
	consumer.Metrics
	consumer.Logs
	xconsumer.Profiles
	component.StartFunc
	component.ShutdownFunc
}

func (c *forward) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}
