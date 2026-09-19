// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type traceContextKeyType int

const (
	batchSpanLinksKey traceContextKeyType = iota
	batchMaxDeadlineKey
)

// LinksFromContext returns a list of trace links registered in the context.
func LinksFromContext(ctx context.Context) []trace.Link {
	if ctx == nil {
		return []trace.Link{}
	}
	if links, ok := ctx.Value(batchSpanLinksKey).([]trace.Link); ok {
		return links
	}
	return []trace.Link{}
}

func parentsFromContext(ctx context.Context) []trace.Link {
	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		return []trace.Link{{SpanContext: spanCtx}}
	}
	return LinksFromContext(ctx)
}

func contextWithMergedLinks(mergedCtx, ctx1, ctx2 context.Context) context.Context {
	return context.WithValue(
		mergedCtx,
		batchSpanLinksKey,
		append(parentsFromContext(ctx1), parentsFromContext(ctx2)...),
	)
}

// deadlineFromContext return the maximum deadline stored during batch context
// merging, and a boolean indicating whether a deadline was stored.
func deadlineFromContext(ctx context.Context) (time.Time, bool) {
	if ctx == nil {
		return time.Time{}, false
	}
	if deadline, ok := ctx.Value(batchMaxDeadlineKey).(time.Time); ok {
		return deadline, true
	}
	return time.Time{}, false
}

// deadlineOf return the effective deadline of a context. It checks the stored
// batch max deadline value first, falling back to the native context deadline
func deadlineOf(ctx context.Context) (time.Time, bool) {
	if ctx == nil {
		return time.Time{}, false
	}
	if d, ok := ctx.Value(batchMaxDeadlineKey).(time.Time); ok {
		return d, true
	}
	return ctx.Deadline()
}

// contextWithMergedDeadline computes the maximum deadline from ctx1 and ctx2,
// and stores it as a context value on mergedCtx.
func contextWithMergedDeadline(mergedCtx, ctx1, ctx2 context.Context) context.Context {
	d1, ok1 := deadlineOf(ctx1)
	d2, ok2 := deadlineOf(ctx2)

	if !ok1 && !ok2 {
		return mergedCtx
	}

	var maxDeadline time.Time
	switch {
	case ok1 && ok2:
		if d1.After(d2) {
			maxDeadline = d1
		} else {
			maxDeadline = d2
		}
	case ok1:
		maxDeadline = d1
	default:
		maxDeadline = d2
	}

	return context.WithValue(mergedCtx, batchMaxDeadlineKey, maxDeadline)
}
