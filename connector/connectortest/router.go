// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connectortest // import "go.opentelemetry.io/collector/connector/connectortest"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

// Deprecated: [v0.92.0] use connector.NewTracesRouter.
type TracesRouterOption struct {
	id   component.ID
	cons consumer.Traces
}

// Deprecated: [v0.92.0] use connector.NewTracesRouter.
func WithNopTraces(id component.ID) TracesRouterOption {
	return TracesRouterOption{id: id, cons: consumertest.NewNop()}
}

// Deprecated: [v0.92.0] use connector.NewTracesRouter.
func WithTracesSink(id component.ID, sink *consumertest.TracesSink) TracesRouterOption {
	return TracesRouterOption{id: id, cons: sink}
}

// Deprecated: [v0.92.0] use connector.NewTracesRouter.
func NewTracesRouter(opts ...TracesRouterOption) connector.TracesRouter {
	consumers := make(map[component.ID]consumer.Traces)
	for _, opt := range opts {
		consumers[opt.id] = opt.cons
	}
	return connector.NewTracesRouter(consumers)
}

// Deprecated: [v0.92.0] use connector.NewMetricsRouter.
type MetricsRouterOption struct {
	id   component.ID
	cons consumer.Metrics
}

// Deprecated: [v0.92.0] use connector.NewMetricsRouter.
func WithNopMetrics(id component.ID) MetricsRouterOption {
	return MetricsRouterOption{id: id, cons: consumertest.NewNop()}
}

// Deprecated: [v0.92.0] use connector.NewMetricsRouter.
func WithMetricsSink(id component.ID, sink *consumertest.MetricsSink) MetricsRouterOption {
	return MetricsRouterOption{id: id, cons: sink}
}

// Deprecated: [v0.92.0] use connector.NewMetricsRouter.
func NewMetricsRouter(opts ...MetricsRouterOption) connector.MetricsRouter {
	consumers := make(map[component.ID]consumer.Metrics)
	for _, opt := range opts {
		consumers[opt.id] = opt.cons
	}
	return connector.NewMetricsRouter(consumers)
}

// Deprecated: [v0.92.0] use connector.NewLogsRouter.
type LogsRouterOption struct {
	id   component.ID
	cons consumer.Logs
}

// Deprecated: [v0.92.0] use connector.NewLogsRouter.
func WithNopLogs(id component.ID) LogsRouterOption {
	return LogsRouterOption{id: id, cons: consumertest.NewNop()}
}

// Deprecated: [v0.92.0] use connector.NewLogsRouter.
func WithLogsSink(id component.ID, sink *consumertest.LogsSink) LogsRouterOption {
	return LogsRouterOption{id: id, cons: sink}
}

// Deprecated: [v0.92.0] use connector.NewLogsRouter.
func NewLogsRouter(opts ...LogsRouterOption) connector.LogsRouter {
	consumers := make(map[component.ID]consumer.Logs)
	for _, opt := range opts {
		consumers[opt.id] = opt.cons
	}
	return connector.NewLogsRouter(consumers)
}
