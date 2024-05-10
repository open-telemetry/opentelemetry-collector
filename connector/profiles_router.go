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

// ProfilesRouterAndConsumer feeds the first consumer.Profiles in each of the specified pipelines.
type ProfilesRouterAndConsumer interface {
	consumer.Profiles
	Consumer(...component.ID) (consumer.Profiles, error)
	PipelineIDs() []component.ID
	privateFunc()
}

type profilesRouter struct {
	consumer.Profiles
	baseRouter[consumer.Profiles]
}

func NewProfilesRouter(cm map[component.ID]consumer.Profiles) ProfilesRouterAndConsumer {
	consumers := make([]consumer.Profiles, 0, len(cm))
	for _, cons := range cm {
		consumers = append(consumers, cons)
	}
	return &profilesRouter{
		Profiles:   fanoutconsumer.NewProfiles(consumers),
		baseRouter: newBaseRouter(fanoutconsumer.NewProfiles, cm),
	}
}

func (r *profilesRouter) PipelineIDs() []component.ID {
	ids := make([]component.ID, 0, len(r.consumers))
	for id := range r.consumers {
		ids = append(ids, id)
	}
	return ids
}

func (r *profilesRouter) Consumer(pipelineIDs ...component.ID) (consumer.Profiles, error) {
	if len(pipelineIDs) == 0 {
		return nil, fmt.Errorf("missing consumers")
	}
	consumers := make([]consumer.Profiles, 0, len(pipelineIDs))
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
		// TODO potentially this could return a NewProfiles with the valid consumers
		return nil, errors
	}
	return fanoutconsumer.NewProfiles(consumers), nil
}

func (r *profilesRouter) privateFunc() {}
