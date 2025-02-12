// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batcher // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/batcher"
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
	return []trace.Link{trace.LinkFromContext(ctx)}
}

func contextWithMergedLinks(ctx1 context.Context, ctx2 context.Context) context.Context {
	return context.WithValue(
		context.Background(),
		batchSpanLinksKey,
		append(LinksFromContext(ctx1), LinksFromContext(ctx2)...))
}
