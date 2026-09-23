// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type traceContextKeyType int

const batchSpanLinksKey traceContextKeyType = iota

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

// contextWithMergedDeadline computes the maximum (latest) deadline from ctx1 and ctx2,
// and applies it to mergedCtx via context.WithDeadline. If neither context has a deadline,
// the context is returned unchanged with a no-op cancel function.
func contextWithMergedDeadline(mergedCtx, ctx1, ctx2 context.Context) (context.Context, context.CancelFunc) {
	var maxDeadline time.Time
	if d, ok := ctx1.Deadline(); ok && d.After(maxDeadline) {
		maxDeadline = d
	}
	if d, ok := ctx2.Deadline(); ok && d.After(maxDeadline) {
		maxDeadline = d
	}

	if maxDeadline.IsZero() {
		return mergedCtx, func() {}
	}

	return context.WithDeadline(mergedCtx, maxDeadline)
}
