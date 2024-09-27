// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"fmt"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/connector/internal"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
	"go.opentelemetry.io/collector/pipeline"
)

// EntitiesRouterAndConsumer feeds the first consumer.Entities in each of the specified pipelines.
type EntitiesRouterAndConsumer interface {
	consumer.Entities
	Consumer(...pipeline.ID) (consumer.Entities, error)
	PipelineIDs() []pipeline.ID
	privateFunc()
}

type entitiesRouter struct {
	consumer.Entities
	internal.BaseRouter[consumer.Entities]
}

func NewEntitiesRouter(cm map[pipeline.ID]consumer.Entities) EntitiesRouterAndConsumer {
	consumers := make([]consumer.Entities, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &entitiesRouter{
		Entities:   fanoutconsumer.NewEntities(consumers),
		BaseRouter: internal.NewBaseRouter(fanoutconsumer.NewEntities, cm),
	}
}

func (r *entitiesRouter) PipelineIDs() []pipeline.ID {
	ids := make([]pipeline.ID, 0, len(r.Consumers))
	for id := range r.Consumers {
		ids = append(ids, id)
	}
	return ids
}

func (r *entitiesRouter) Consumer(pipelineIDs ...pipeline.ID) (consumer.Entities, error) {
	if len(pipelineIDs) == 0 {
		return nil, fmt.Errorf("missing consumers")
	}
	consumers := make([]consumer.Entities, 0, len(pipelineIDs))
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
		// TODO potentially this could return a NewEntities with the valid consumers
		return nil, errors
	}
	return fanoutconsumer.NewEntities(consumers), nil
}

func (r *entitiesRouter) privateFunc() {}
