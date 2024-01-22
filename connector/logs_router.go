// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"fmt"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
)

// Deprecated: [v0.92.0] use LogsRouterAndConsumer
type LogsRouter interface {
	Consumer(...component.ID) (consumer.Logs, error)
	PipelineIDs() []component.ID
}

// LogsRouterAndConsumer feeds the first consumer.Logs in each of the specified pipelines.
type LogsRouterAndConsumer interface {
	consumer.Logs
	Consumer(...component.ID) (consumer.Logs, error)
	PipelineIDs() []component.ID
	privateFunc()
}

type logsRouter struct {
	consumer.Logs
	consumers map[component.ID]consumer.Logs
}

func NewLogsRouter(cm map[component.ID]consumer.Logs) LogsRouterAndConsumer {
	consumers := make([]consumer.Logs, 0, len(cm))
	for _, consumer := range cm {
		consumers = append(consumers, consumer)
	}
	return &logsRouter{
		Logs:      fanoutconsumer.NewLogs(consumers),
		consumers: cm,
	}
}

func (r *logsRouter) PipelineIDs() []component.ID {
	ids := make([]component.ID, 0, len(r.consumers))
	for id := range r.consumers {
		ids = append(ids, id)
	}
	return ids
}

func (r *logsRouter) Consumer(pipelineIDs ...component.ID) (consumer.Logs, error) {
	if len(pipelineIDs) == 0 {
		return nil, fmt.Errorf("missing consumers")
	}
	consumers := make([]consumer.Logs, 0, len(pipelineIDs))
	var errors error
	for _, pipelineID := range pipelineIDs {
		c, ok := r.consumers[pipelineID]
		if ok {
			consumers = append(consumers, c)
		} else {
			errors = multierr.Append(errors, fmt.Errorf("missing consumer: %q", pipelineID))
		}
	}
	if errors != nil {
		// TODO potentially this could return a NewLogs with the valid consumers
		return nil, errors
	}
	return fanoutconsumer.NewLogs(consumers), nil
}

func (r *logsRouter) privateFunc() {}
