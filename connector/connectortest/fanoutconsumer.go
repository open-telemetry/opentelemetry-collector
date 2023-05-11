// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package connectortest // import "go.opentelemetry.io/collector/connector/connectortest"

import (
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
)

var errTooFewConsumers = errors.New("fanoutconsumer requires a mininum of 2 consumers")

type TracesRouterTestOption struct {
	id   component.ID
	cons consumer.Traces
}

// WithNopTracesSink creates a nop consumer for a connector.TracesRouter
func WithNopTracesSink(id component.ID) TracesRouterTestOption {
	return TracesRouterTestOption{id: id, cons: consumertest.NewNop()}
}

// WithTracesSink adds a consumer to a connector.TracesRouter
func WithTracesSink(id component.ID, sink *consumertest.TracesSink) TracesRouterTestOption {
	return TracesRouterTestOption{id: id, cons: sink}
}

// NewTracesRouterSink returns a connector.TracesRouter with sinks based on the options provided
func NewTracesRouterSink(opts ...TracesRouterTestOption) (connector.TracesRouter, error) {
	consumers := make(map[component.ID]consumer.Traces)
	for _, opt := range opts {
		consumers[opt.id] = opt.cons
	}
	if len(consumers) < 2 {
		return nil, errTooFewConsumers
	}
	return fanoutconsumer.NewTracesRouter(consumers).(connector.TracesRouter), nil
}

type MetricsRouterTestOption struct {
	id   component.ID
	cons consumer.Metrics
}

// WithNopMetricsSink creates a nop consumer for a connector.MetricsRouter
func WithNopMetricsSink(id component.ID) MetricsRouterTestOption {
	return MetricsRouterTestOption{id: id, cons: consumertest.NewNop()}
}

// WithMetricsSink adds a consumer to a connector.MetricsRouter
func WithMetricsSink(id component.ID, sink *consumertest.MetricsSink) MetricsRouterTestOption {
	return MetricsRouterTestOption{id: id, cons: sink}
}

// NewMetricsRouterSink returns a connector.MetricsRouter with sinks based on the options provided
func NewMetricsRouterSink(opts ...MetricsRouterTestOption) (connector.MetricsRouter, error) {
	consumers := make(map[component.ID]consumer.Metrics)
	for _, opt := range opts {
		consumers[opt.id] = opt.cons
	}
	if len(consumers) < 2 {
		return nil, errTooFewConsumers
	}
	return fanoutconsumer.NewMetricsRouter(consumers).(connector.MetricsRouter), nil
}

type LogsRouterTestOption struct {
	id   component.ID
	cons consumer.Logs
}

// WithNopLogsSink creates a nop consumer for a connector.LogsRouter
func WithNopLogsSink(id component.ID) LogsRouterTestOption {
	return LogsRouterTestOption{id: id, cons: consumertest.NewNop()}
}

// WithLogsSink adds a consumer to a connector.LogsRouter
func WithLogsSink(id component.ID, sink *consumertest.LogsSink) LogsRouterTestOption {
	return LogsRouterTestOption{id: id, cons: sink}
}

// NewLogsRouterSink returns a connector.LogsRouter with sinks based on the options provided
func NewLogsRouterSink(opts ...LogsRouterTestOption) (connector.LogsRouter, error) {
	consumers := make(map[component.ID]consumer.Logs)
	for _, opt := range opts {
		consumers[opt.id] = opt.cons
	}
	if len(consumers) < 2 {
		return nil, errTooFewConsumers
	}
	return fanoutconsumer.NewLogsRouter(consumers).(connector.LogsRouter), nil
}
