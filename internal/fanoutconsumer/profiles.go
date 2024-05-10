// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package fanoutconsumer contains implementations of Traces/Metrics/Profiles consumers
// that fan out the data to multiple other consumers.
package fanoutconsumer // import "go.opentelemetry.io/collector/service/internal/fanoutconsumer"

import (
	"context"
	"fmt"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

// NewProfiles wraps multiple profile consumers in a single one.
// It fanouts the incoming data to all the consumers, and does smart routing:
//   - Clones only to the consumer that needs to mutate the data.
//   - If all consumers needs to mutate the data one will get the original data.
func NewProfiles(lcs []consumer.Profiles) consumer.Profiles {
	if len(lcs) == 1 {
		// Don't wrap if no need to do it.
		return lcs[0]
	}
	var pass []consumer.Profiles
	var clone []consumer.Profiles
	for i := 0; i < len(lcs)-1; i++ {
		if !lcs[i].Capabilities().MutatesData {
			pass = append(pass, lcs[i])
		} else {
			clone = append(clone, lcs[i])
		}
	}
	// Give the original data to the last consumer if no other read-only consumer,
	// otherwise put it in the right bucket. Never share the same data between
	// a mutating and a non-mutating consumer since the non-mutating consumer may process
	// data async and the mutating consumer may change the data before that.
	if len(pass) == 0 || !lcs[len(lcs)-1].Capabilities().MutatesData {
		pass = append(pass, lcs[len(lcs)-1])
	} else {
		clone = append(clone, lcs[len(lcs)-1])
	}
	return &profilesConsumer{pass: pass, clone: clone}
}

type profilesConsumer struct {
	pass  []consumer.Profiles
	clone []consumer.Profiles
}

func (lsc *profilesConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeProfiles exports the pprofile.Profiles to all consumers wrapped by the current one.
func (lsc *profilesConsumer) ConsumeProfiles(ctx context.Context, ld pprofile.Profiles) error {
	var errs error
	// Initially pass to clone exporter to avoid the case where the optimization of sending
	// the incoming data to a mutating consumer is used that may change the incoming data before
	// cloning.
	for _, lc := range lsc.clone {
		clonedProfiles := pprofile.NewProfiles()
		ld.CopyTo(clonedProfiles)
		errs = multierr.Append(errs, lc.ConsumeProfiles(ctx, clonedProfiles))
	}
	for _, lc := range lsc.pass {
		errs = multierr.Append(errs, lc.ConsumeProfiles(ctx, ld))
	}
	return errs
}

var _ connector.ProfilesRouter = (*profilesRouter)(nil)

type profilesRouter struct {
	consumer.Profiles
	consumers map[component.ID]consumer.Profiles
}

func NewProfilesRouter(cm map[component.ID]consumer.Profiles) consumer.Profiles {
	consumers := make([]consumer.Profiles, 0, len(cm))
	for _, consumer := range cm {
		consumers = append(consumers, consumer)
	}
	return &profilesRouter{
		Profiles:  NewProfiles(consumers),
		consumers: cm,
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
	return NewProfiles(consumers), nil
}
