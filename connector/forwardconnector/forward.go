// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package forwardconnector // import "go.opentelemetry.io/collector/connector/forwardconnector"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
)

const (
	typeStr = "forward"
)

// NewFactory returns a connector.Factory.
func NewFactory() connector.Factory {
	return connector.NewFactory(
		typeStr,
		createDefaultConfig,
		connector.WithTracesToTraces(createTracesToTraces, component.StabilityLevelBeta),
		connector.WithMetricsToMetrics(createMetricsToMetrics, component.StabilityLevelBeta),
		connector.WithLogsToLogs(createLogsToLogs, component.StabilityLevelBeta),
	)
}

// createDefaultConfig creates the default configuration.
func createDefaultConfig() component.Config {
	return &struct{}{}
}

// createTracesToTraces creates a trace receiver based on provided config.
func createTracesToTraces(
	_ context.Context,
	_ connector.CreateSettings,
	_ component.Config,
	nextConsumer consumer.Traces,
) (connector.Traces, error) {
	return &forward{Traces: nextConsumer}, nil
}

// createMetricsToMetrics creates a metrics receiver based on provided config.
func createMetricsToMetrics(
	_ context.Context,
	_ connector.CreateSettings,
	_ component.Config,
	nextConsumer consumer.Metrics,
) (connector.Metrics, error) {
	return &forward{Metrics: nextConsumer}, nil
}

// createLogsToLogs creates a log receiver based on provided config.
func createLogsToLogs(
	_ context.Context,
	_ connector.CreateSettings,
	_ component.Config,
	nextConsumer consumer.Logs,
) (connector.Logs, error) {
	return &forward{Logs: nextConsumer}, nil
}

// forward is used to pass signals directly from one pipeline to another.
// This is useful when there is a need to replicate data and process it in more
// than one way. It can also be used to join pipelines together.
type forward struct {
	consumer.Traces
	consumer.Metrics
	consumer.Logs
	component.StartFunc
	component.ShutdownFunc
}

func (c *forward) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}
