// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batcher // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/batcher"
import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// mergedContext links the underlying context to all incoming span contexts.
type mergedContext struct {
	deadline   time.Time
	deadlineOk bool
	ctx        context.Context
}

func NewMergedContext(ctx context.Context) mergedContext {
	deadline, ok := ctx.Deadline()
	return mergedContext{
		deadline:   deadline,
		deadlineOk: ok,
		ctx:        ctx,
	}
}

// Merge links the span from incoming context to the span from the first context.
func (mc *mergedContext) Merge(other context.Context) mergedContext {
	deadline, deadlineOk := mc.Deadline()
	if otherDeadline, ok := other.Deadline(); ok {
		deadlineOk = true
		if deadline.Before(otherDeadline) {
			deadline = otherDeadline
		}
	}

	link := trace.LinkFromContext(other)
	trace.SpanFromContext(mc.ctx).AddLink(link)
	return mergedContext{
		deadline:   deadline,
		deadlineOk: deadlineOk,
		ctx:        mc.ctx,
	}
}

// Deadline returns the latest deadline of all context.
func (mc mergedContext) Deadline() (time.Time, bool) {
	return mc.deadline, mc.deadlineOk
}

func (mc mergedContext) Done() <-chan struct{} {
	return nil
}

func (mc mergedContext) Err() error {
	return nil
}

// Value delegates to the first context.
func (mc mergedContext) Value(key any) any {
	return mc.ctx.Value(key)
}
