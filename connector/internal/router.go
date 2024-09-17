// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/connector/internal"

import (
	"fmt"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
)

type BaseRouter[T any] struct {
	fanout    func([]T) T
	Consumers map[component.ID]T
}

func NewBaseRouter[T any](fanout func([]T) T, cm map[component.ID]T) BaseRouter[T] {
	consumers := make(map[component.ID]T, len(cm))
	for k, v := range cm {
		consumers[k] = v
	}
	return BaseRouter[T]{fanout: fanout, Consumers: consumers}
}

func (r *BaseRouter[T]) PipelineIDs() []component.ID {
	ids := make([]component.ID, 0, len(r.Consumers))
	for id := range r.Consumers {
		ids = append(ids, id)
	}
	return ids
}

func (r *BaseRouter[T]) Consumer(pipelineIDs ...component.ID) (T, error) {
	var ret T
	if len(pipelineIDs) == 0 {
		return ret, fmt.Errorf("missing consumers")
	}
	consumers := make([]T, 0, len(pipelineIDs))
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
		// TODO potentially this could return a NewTraces with the valid consumers
		return ret, errors
	}
	return r.fanout(consumers), nil
}

type BaseRouterWithPipelineIDs[T any] struct {
	fanout    func([]T) T
	Consumers map[pipeline.ID]T
}

func NewBaseRouterWithPipelineIDs[T any](fanout func([]T) T, cm map[pipeline.ID]T) BaseRouterWithPipelineIDs[T] {
	consumers := make(map[pipeline.ID]T, len(cm))
	for k, v := range cm {
		consumers[k] = v
	}
	return BaseRouterWithPipelineIDs[T]{fanout: fanout, Consumers: consumers}
}

func (r *BaseRouterWithPipelineIDs[T]) PipelineIDs() []pipeline.ID {
	ids := make([]pipeline.ID, 0, len(r.Consumers))
	for id := range r.Consumers {
		ids = append(ids, id)
	}
	return ids
}

func (r *BaseRouterWithPipelineIDs[T]) Consumer(pipelineIDs ...pipeline.ID) (T, error) {
	var ret T
	if len(pipelineIDs) == 0 {
		return ret, fmt.Errorf("missing consumers")
	}
	consumers := make([]T, 0, len(pipelineIDs))
	var errors error
	for _, pipelineID := range pipelineIDs {
		c, ok := r.Consumers[pipelineID]
		if ok {
			consumers = append(consumers, c)
		} else {
			errors = multierr.Append(errors, fmt.Errorf("missing consumer: %q", pipelineID.String()))
		}
	}
	if errors != nil {
		// TODO potentially this could return a NewTraces with the valid consumers
		return ret, errors
	}
	return r.fanout(consumers), nil
}
