// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package fanoutconsumer contains implementations of Traces/Metrics/Entities consumers
// that fan out the data to multiple other consumers.
package fanoutconsumer // import "go.opentelemetry.io/collector/internal/fanoutconsumer"

import (
	"context"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/pentity"
)

// NewEntities wraps multiple log consumers in a single one.
// It fans out the incoming data to all the consumers, and does smart routing:
//   - Clones only to the consumer that needs to mutate the data.
//   - If all consumers needs to mutate the data one will get the original mutable data.
func NewEntities(lcs []xconsumer.Entities) xconsumer.Entities {
	// Don't wrap if there is only one non-mutating consumer.
	if len(lcs) == 1 && !lcs[0].Capabilities().MutatesData {
		return lcs[0]
	}

	lc := &entitiesConsumer{}
	for i := 0; i < len(lcs); i++ {
		if lcs[i].Capabilities().MutatesData {
			lc.mutable = append(lc.mutable, lcs[i])
		} else {
			lc.readonly = append(lc.readonly, lcs[i])
		}
	}
	return lc
}

type entitiesConsumer struct {
	mutable  []xconsumer.Entities
	readonly []xconsumer.Entities
}

func (lsc *entitiesConsumer) Capabilities() consumer.Capabilities {
	// If all consumers are mutating, then the original data will be passed to one of them.
	return consumer.Capabilities{MutatesData: len(lsc.mutable) > 0 && len(lsc.readonly) == 0}
}

// ConsumeEntities exports the pentity.Entities to all consumers wrapped by the current one.
func (lsc *entitiesConsumer) ConsumeEntities(ctx context.Context, ld pentity.Entities) error {
	var errs error

	if len(lsc.mutable) > 0 {
		// Clone the data before sending to all mutating consumers except the last one.
		for i := 0; i < len(lsc.mutable)-1; i++ {
			errs = multierr.Append(errs, lsc.mutable[i].ConsumeEntities(ctx, cloneEntities(ld)))
		}
		// Send data as is to the last mutating consumer only if there are no other non-mutating consumers and the
		// data is mutable. Never share the same data between a mutating and a non-mutating consumer since the
		// non-mutating consumer may process data async and the mutating consumer may change the data before that.
		lastConsumer := lsc.mutable[len(lsc.mutable)-1]
		if len(lsc.readonly) == 0 && !ld.IsReadOnly() {
			errs = multierr.Append(errs, lastConsumer.ConsumeEntities(ctx, ld))
		} else {
			errs = multierr.Append(errs, lastConsumer.ConsumeEntities(ctx, cloneEntities(ld)))
		}
	}

	// Mark the data as read-only if it will be sent to more than one read-only consumer.
	if len(lsc.readonly) > 1 && !ld.IsReadOnly() {
		ld.MarkReadOnly()
	}
	for _, lc := range lsc.readonly {
		errs = multierr.Append(errs, lc.ConsumeEntities(ctx, ld))
	}

	return errs
}

func cloneEntities(ld pentity.Entities) pentity.Entities {
	clonedEntities := pentity.NewEntities()
	ld.CopyTo(clonedEntities)
	return clonedEntities
}
