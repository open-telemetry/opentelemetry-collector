// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"fmt"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/clog"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
)

// LogsRouterAndConsumer feeds the first clog.Logs in each of the specified pipelines.
type LogsRouterAndConsumer interface {
	clog.Logs
	Consumer(...component.ID) (clog.Logs, error)
	PipelineIDs() []component.ID
	privateFunc()
}

type logsRouter struct {
	clog.Logs
	baseRouter[clog.Logs]
}

func NewLogsRouter(cm map[component.ID]clog.Logs) LogsRouterAndConsumer {
	consumers := make([]clog.Logs, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &logsRouter{
		Logs:       fanoutconsumer.NewLogs(consumers),
		baseRouter: newBaseRouter(fanoutconsumer.NewLogs, cm),
	}
}

func (r *logsRouter) PipelineIDs() []component.ID {
	ids := make([]component.ID, 0, len(r.consumers))
	for id := range r.consumers {
		ids = append(ids, id)
	}
	return ids
}

func (r *logsRouter) Consumer(pipelineIDs ...component.ID) (clog.Logs, error) {
	if len(pipelineIDs) == 0 {
		return nil, fmt.Errorf("missing consumers")
	}
	consumers := make([]clog.Logs, 0, len(pipelineIDs))
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
