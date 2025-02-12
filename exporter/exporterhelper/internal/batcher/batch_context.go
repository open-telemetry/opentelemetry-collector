// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batcher // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/batcher"
import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type traceContextKeyType int

const batchSpanLinksKey traceContextKeyType = iota

// batchContext links the underlying context to all incoming span contexts.
type batchContext struct {
	deadline    time.Time
	deadlineOk  bool
	deadlineCtx context.Context
	ctx         context.Context
}

func SpanLinksFromContext(ctx context.Context) []trace.Link {
	if ctx == nil {
		return []trace.Link{}
	}

	if bctx, ok := ctx.(batchContext); ok {
		if links, ok := bctx.Value(batchSpanLinksKey).([]trace.Link); ok {
			return links
		}
	} else {
		panic("lalal")
	}
	return []trace.Link{}
}

func newBatchContext(ctx context.Context) batchContext {
	deadline, ok := ctx.Deadline()
	underlyingCtx := context.WithValue(
		context.Background(),
		batchSpanLinksKey,
		[]trace.Link{trace.LinkFromContext(ctx)})
	return batchContext{
		deadline:    deadline,
		deadlineOk:  ok,
		deadlineCtx: ctx,
		ctx:         underlyingCtx,
	}
}

// Merge links the span from incoming context to the span from the first context.
func (mc batchContext) Merge(other context.Context) batchContext {
	deadline, deadlineOk := mc.Deadline()
	deadlineCtx := mc.deadlineCtx
	if otherDeadline, ok := other.Deadline(); ok {
		deadlineOk = true
		if deadline.Before(otherDeadline) {
			deadline = otherDeadline
			deadlineCtx = other
		}
	}

	links := append(SpanLinksFromContext(mc), trace.LinkFromContext(other))
	underlyingCtx := context.WithValue(
		mc.ctx,
		batchSpanLinksKey,
		links)
	return batchContext{
		deadline:    deadline,
		deadlineOk:  deadlineOk,
		deadlineCtx: deadlineCtx,
		ctx:         underlyingCtx,
	}
}

// Deadline returns the latest deadline of all context.
func (mc batchContext) Deadline() (time.Time, bool) {
	return mc.deadline, mc.deadlineOk
}

func (mc batchContext) Done() <-chan struct{} {
	return mc.deadlineCtx.Done()
}

func (mc batchContext) Err() error {
	return mc.deadlineCtx.Err()
}

func (mc batchContext) Value(key any) any {
	return mc.ctx.Value(key)
}
