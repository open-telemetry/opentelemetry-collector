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
	deadline    time.Time
	deadlineOk  bool
	deadlineCtx context.Context
	ctx         context.Context
}

func newMergedContext(ctx context.Context) mergedContext {
	deadline, ok := ctx.Deadline()
	return mergedContext{
		deadline:    deadline,
		deadlineOk:  ok,
		deadlineCtx: ctx,
		ctx:         ctx,
	}
}

// Merge links the span from incoming context to the span from the first context.
func (mc *mergedContext) Merge(other context.Context) mergedContext {
	deadline, deadlineOk := mc.Deadline()
	ctx := mc.deadlineCtx
	if otherDeadline, ok := other.Deadline(); ok {
		deadlineOk = true
		if deadline.Before(otherDeadline) {
			deadline = otherDeadline
			ctx = other
		}
	}

	link := trace.LinkFromContext(other)
	trace.SpanFromContext(mc.ctx).AddLink(link)
	return mergedContext{
		deadline:    deadline,
		deadlineOk:  deadlineOk,
		deadlineCtx: ctx,
		ctx:         mc.ctx,
	}
}

// Deadline returns the latest deadline of all context.
func (mc mergedContext) Deadline() (time.Time, bool) {
	return mc.deadline, mc.deadlineOk
}

func (mc mergedContext) Done() <-chan struct{} {
	return mc.deadlineCtx.Done()
}

func (mc mergedContext) Err() error {
	return mc.deadlineCtx.Err()
}

// Value delegates to the first context.
func (mc mergedContext) Value(key any) any {
	return mc.ctx.Value(key)
}
