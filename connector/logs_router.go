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

// LogsRouterAndConsumer feeds the first consumer.Logs in each of the specified pipelines.
type LogsRouterAndConsumer interface {
	consumer.Logs
	Consumer(...component.DataTypeID) (consumer.Logs, error)
	PipelineIDs() []component.DataTypeID
	privateFunc()
}

type logsRouter struct {
	consumer.Logs
	baseRouter[consumer.Logs]
}

func NewLogsRouter(cm map[component.DataTypeID]consumer.Logs) LogsRouterAndConsumer {
	consumers := make([]consumer.Logs, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &logsRouter{
		Logs:       fanoutconsumer.NewLogs(consumers),
		baseRouter: newBaseRouter(fanoutconsumer.NewLogs, cm),
	}
}

func (r *logsRouter) PipelineIDs() []component.DataTypeID {
	ids := make([]component.DataTypeID, 0, len(r.consumers))
	for id := range r.consumers {
		ids = append(ids, id)
	}
	return ids
}

func (r *logsRouter) Consumer(pipelineIDs ...component.DataTypeID) (consumer.Logs, error) {
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
