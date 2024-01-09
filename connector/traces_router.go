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

// Deprecated: [v0.92.0] use TracesRouterAndConsumer
type TracesRouter interface {
	Consumer(...component.ID) (consumer.Traces, error)
	PipelineIDs() []component.ID
}

// TracesRouterAndConsumer feeds the first consumer.Traces in each of the specified pipelines.
type TracesRouterAndConsumer interface {
	consumer.Traces
	Consumer(...component.ID) (consumer.Traces, error)
	PipelineIDs() []component.ID
	privateFunc()
}

type tracesRouter struct {
	consumer.Traces
	consumers map[component.ID]consumer.Traces
}

func NewTracesRouter(cm map[component.ID]consumer.Traces) TracesRouterAndConsumer {
	consumers := make([]consumer.Traces, 0, len(cm))
	for _, c := range cm {
		consumers = append(consumers, c)
	}
	return &tracesRouter{
		Traces:    fanoutconsumer.NewTraces(consumers),
		consumers: cm,
	}
}

func (r *tracesRouter) PipelineIDs() []component.ID {
	ids := make([]component.ID, 0, len(r.consumers))
	for id := range r.consumers {
		ids = append(ids, id)
	}
	return ids
}

func (r *tracesRouter) Consumer(pipelineIDs ...component.ID) (consumer.Traces, error) {
	if len(pipelineIDs) == 0 {
		return nil, fmt.Errorf("missing consumers")
	}
	consumers := make([]consumer.Traces, 0, len(pipelineIDs))
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
		// TODO potentially this could return a NewTraces with the valid consumers
		return nil, errors
	}
	return fanoutconsumer.NewTraces(consumers), nil
}

func (r *tracesRouter) privateFunc() {}
