// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"errors"
	"fmt"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/connector/internal"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
	"go.opentelemetry.io/collector/pipeline"
)

// LogsRouterAndConsumer feeds the first consumer.Logs in each of the specified pipelines.
type LogsRouterAndConsumer interface {
	consumer.Logs
	Consumer(...pipeline.ID) (consumer.Logs, error)
	PipelineIDs() []pipeline.ID
	privateFunc()
}

type logsRouter struct {
	consumer.Logs
	internal.BaseRouter[consumer.Logs]
}

func NewLogsRouter(cm map[pipeline.ID]consumer.Logs) LogsRouterAndConsumer {
	consumers := make([]consumer.Logs, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &logsRouter{
		Logs:       fanoutconsumer.NewLogs(consumers),
		BaseRouter: internal.NewBaseRouter(fanoutconsumer.NewLogs, cm),
	}
}

func (r *logsRouter) PipelineIDs() []pipeline.ID {
	ids := make([]pipeline.ID, 0, len(r.Consumers))
	for id := range r.Consumers {
		ids = append(ids, id)
	}
	return ids
}

func (r *logsRouter) Consumer(pipelineIDs ...pipeline.ID) (consumer.Logs, error) {
	if len(pipelineIDs) == 0 {
		return nil, errors.New("missing consumers")
	}
	consumers := make([]consumer.Logs, 0, len(pipelineIDs))
	var errors error
	for _, pipelineID := range pipelineIDs {
		c, ok := r.Consumers[pipelineID]
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
