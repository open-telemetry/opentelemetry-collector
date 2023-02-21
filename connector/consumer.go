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

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/fanoutconsumer"
)

// TracesConsumer feeds the first consumer.Traces in each of the specified pipelines.
// Unlike other consumers, the connector consumer will expose its underlying consumers.
type TracesConsumer interface {
	consumer.Traces
	Consumers() map[component.ID]consumer.Traces
}

var _ TracesConsumer = (*tracesConsumer)(nil)

type tracesConsumer struct {
	consumer.ConsumeTracesFunc
	consumers map[component.ID]consumer.Traces
}

func NewTracesConsumer(cm map[component.ID]consumer.Traces) TracesConsumer {
	consumers := make([]consumer.Traces, 0, len(cm))
	for _, consumer := range cm {
		consumers = append(consumers, consumer)
	}
	return &tracesConsumer{
		ConsumeTracesFunc: fanoutconsumer.NewTraces(consumers).ConsumeTraces,
		consumers:         cm,
	}
}

func (r *tracesConsumer) Consumers() map[component.ID]consumer.Traces {
	consumers := make(map[component.ID]consumer.Traces, len(r.consumers))
	for k, v := range r.consumers {
		consumers[k] = v
	}
	return consumers
}

func (r *tracesConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// MetricsConsumer feeds the first consumer.Metrics in each of the specified pipelines.
// Unlike other consumers, the connector consumer will expose its underlying consumers.
type MetricsConsumer interface {
	consumer.Metrics
	Consumers() map[component.ID]consumer.Metrics
}

var _ MetricsConsumer = (*metricsConsumer)(nil)

type metricsConsumer struct {
	consumer.ConsumeMetricsFunc
	consumers map[component.ID]consumer.Metrics
}

func NewMetricsConsumer(cm map[component.ID]consumer.Metrics) MetricsConsumer {
	consumers := make([]consumer.Metrics, 0, len(cm))
	for _, consumer := range cm {
		consumers = append(consumers, consumer)
	}
	return &metricsConsumer{
		ConsumeMetricsFunc: fanoutconsumer.NewMetrics(consumers).ConsumeMetrics,
		consumers:          cm,
	}
}

func (r *metricsConsumer) Consumers() map[component.ID]consumer.Metrics {
	consumers := make(map[component.ID]consumer.Metrics, len(r.consumers))
	for k, v := range r.consumers {
		consumers[k] = v
	}
	return consumers
}

func (r *metricsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// LogsConsumer feeds the first consumer.Logs in each of the specified pipelines.
// Unlike other consumers, the connector consumer will expose its underlying consumers.
type LogsConsumer interface {
	consumer.Logs
	Consumers() map[component.ID]consumer.Logs
}

var _ LogsConsumer = (*logsConsumer)(nil)

type logsConsumer struct {
	consumer.ConsumeLogsFunc
	consumers map[component.ID]consumer.Logs
}

func NewLogsConsumer(cm map[component.ID]consumer.Logs) LogsConsumer {
	consumers := make([]consumer.Logs, 0, len(cm))
	for _, consumer := range cm {
		consumers = append(consumers, consumer)
	}
	return &logsConsumer{
		ConsumeLogsFunc: fanoutconsumer.NewLogs(consumers).ConsumeLogs,
		consumers:       cm,
	}
}

func (r *logsConsumer) Consumers() map[component.ID]consumer.Logs {
	consumers := make(map[component.ID]consumer.Logs, len(r.consumers))
	for k, v := range r.consumers {
		consumers[k] = v
	}
	return consumers
}

func (r *logsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}
