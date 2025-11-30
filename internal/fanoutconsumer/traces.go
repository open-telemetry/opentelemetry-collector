// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fanoutconsumer // import "go.opentelemetry.io/collector/internal/fanoutconsumer"

import (
	"context"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// NewTraces wraps multiple trace consumers in a single one.
// It fans out the incoming data to all the consumers, and does smart routing:
//   - Clones only to the consumer that needs to mutate the data.
//   - If all consumers needs to mutate the data one will get the original mutable data.
func NewTraces(tcs []consumer.Traces) consumer.Traces {
	// Don't wrap if there is only one non-mutating consumer.
	if len(tcs) == 1 && !tcs[0].Capabilities().MutatesData {
		return tcs[0]
	}

	tc := &tracesConsumer{}
	for i := range tcs {
		if tcs[i].Capabilities().MutatesData {
			tc.mutable = append(tc.mutable, tcs[i])
		} else {
			tc.readonly = append(tc.readonly, tcs[i])
		}
	}
	return tc
}

type tracesConsumer struct {
	mutable  []consumer.Traces
	readonly []consumer.Traces
}

func (tsc *tracesConsumer) Capabilities() consumer.Capabilities {
	// If all consumers are mutating, then the original data will be passed to one of them.
	return consumer.Capabilities{MutatesData: len(tsc.mutable) > 0 && len(tsc.readonly) == 0}
}

// ConsumeTraces exports the ptrace.Traces to all consumers wrapped by the current one.
func (tsc *tracesConsumer) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	var errs error

	if len(tsc.mutable) > 0 {
		// Clone the data before sending to all mutating consumers except the last one.
		for i := 0; i < len(tsc.mutable)-1; i++ {
			errs = multierr.Append(errs, tsc.mutable[i].ConsumeTraces(ctx, cloneTraces(td)))
		}
		// Send data as is to the last mutating consumer only if there are no other non-mutating consumers and the
		// data is mutable. Never share the same data between a mutating and a non-mutating consumer since the
		// non-mutating consumer may process data async and the mutating consumer may change the data before that.
		lastConsumer := tsc.mutable[len(tsc.mutable)-1]
		if len(tsc.readonly) == 0 && !td.IsReadOnly() {
			errs = multierr.Append(errs, lastConsumer.ConsumeTraces(ctx, td))
		} else {
			errs = multierr.Append(errs, lastConsumer.ConsumeTraces(ctx, cloneTraces(td)))
		}
	}

	// Mark the data as read-only if it will be sent to more than one read-only consumer.
	if len(tsc.readonly) > 1 && !td.IsReadOnly() {
		td.MarkReadOnly()
	}
	for _, tc := range tsc.readonly {
		errs = multierr.Append(errs, tc.ConsumeTraces(ctx, td))
	}

	return errs
}

func cloneTraces(td ptrace.Traces) ptrace.Traces {
	clonedTraces := ptrace.NewTraces()
	td.CopyTo(clonedTraces)
	return clonedTraces
}
