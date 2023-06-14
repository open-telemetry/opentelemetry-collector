// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connectortest // import "go.opentelemetry.io/collector/connector/connectortest"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
)

type TracesRouterOption struct {
	id   component.ID
	cons consumer.Traces
}

// WithNopTraces creates a nop consumer for a connector.TracesRouter
func WithNopTraces(id component.ID) TracesRouterOption {
	return TracesRouterOption{id: id, cons: consumertest.NewNop()}
}

// WithTracesSink adds a consumer to a connector.TracesRouter
func WithTracesSink(id component.ID, sink *consumertest.TracesSink) TracesRouterOption {
	return TracesRouterOption{id: id, cons: sink}
}

// NewTracesRouter returns a connector.TracesRouter with sinks based on the options provided
func NewTracesRouter(opts ...TracesRouterOption) connector.TracesRouter {
	consumers := make(map[component.ID]consumer.Traces)
	for _, opt := range opts {
		consumers[opt.id] = opt.cons
	}
	return fanoutconsumer.NewTracesRouter(consumers).(connector.TracesRouter)
}

type MetricsRouterOption struct {
	id   component.ID
	cons consumer.Metrics
}

// WithNopMetrics creates a nop consumer for a connector.MetricsRouter
func WithNopMetrics(id component.ID) MetricsRouterOption {
	return MetricsRouterOption{id: id, cons: consumertest.NewNop()}
}

// WithMetricsSink adds a consumer to a connector.MetricsRouter
func WithMetricsSink(id component.ID, sink *consumertest.MetricsSink) MetricsRouterOption {
	return MetricsRouterOption{id: id, cons: sink}
}

// NewMetricsRouter returns a connector.MetricsRouter with sinks based on the options provided
func NewMetricsRouter(opts ...MetricsRouterOption) connector.MetricsRouter {
	consumers := make(map[component.ID]consumer.Metrics)
	for _, opt := range opts {
		consumers[opt.id] = opt.cons
	}
	return fanoutconsumer.NewMetricsRouter(consumers).(connector.MetricsRouter)
}

type LogsRouterOption struct {
	id   component.ID
	cons consumer.Logs
}

// WithNopLogs creates a nop consumer for a connector.LogsRouter
func WithNopLogs(id component.ID) LogsRouterOption {
	return LogsRouterOption{id: id, cons: consumertest.NewNop()}
}

// WithLogsSink adds a consumer to a connector.LogsRouter
func WithLogsSink(id component.ID, sink *consumertest.LogsSink) LogsRouterOption {
	return LogsRouterOption{id: id, cons: sink}
}

// NewLogsRouter returns a connector.LogsRouter with sinks based on the options provided
func NewLogsRouter(opts ...LogsRouterOption) connector.LogsRouter {
	consumers := make(map[component.ID]consumer.Logs)
	for _, opt := range opts {
		consumers[opt.id] = opt.cons
	}
	return fanoutconsumer.NewLogsRouter(consumers).(connector.LogsRouter)
}
