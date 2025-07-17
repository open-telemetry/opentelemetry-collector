// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"

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

func contextWithMergedLinks(ctx1 context.Context, ctx2 context.Context) context.Context {
	return mergedContext{
		context.WithValue(context.Background(),
			batchSpanLinksKey,
			append(parentsFromContext(ctx1), parentsFromContext(ctx2)...)),
		ctx1,
		ctx2,
	}
}

type mergedContext struct {
	context.Context
	ctx1 context.Context
	ctx2 context.Context
}

func (c mergedContext) Value(key any) any {
	if c.ctx1 != nil {
		if val := c.ctx1.Value(key); val != nil {
			return val
		}
	}
	if c.ctx2 != nil {
		if val := c.ctx2.Value(key); val != nil {
			return val
		}
	}
	return nil
}
