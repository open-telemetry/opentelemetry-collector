// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/pdata/xpdata/request/internal"
)

// encodeContext encodes the context into a map of strings.
func encodeContext(ctx context.Context) internal.RequestContext {
	rc := internal.RequestContext{}
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		traceID := spanCtx.TraceID()
		spanID := spanCtx.SpanID()
		rc.SpanContext = &internal.SpanContext{
			TraceId:    traceID[:],
			SpanId:     spanID[:],
			TraceFlags: uint32(spanCtx.TraceFlags()),
			TraceState: spanCtx.TraceState().String(),
			Remote:     spanCtx.IsRemote(),
		}
	}
	return rc
}

// decodeContext decodes the context from the bytes map.
func decodeContext(rc *internal.RequestContext) context.Context {
	if rc == nil || rc.SpanContext == nil {
		return context.Background()
	}
	traceID := trace.TraceID{}
	copy(traceID[:], rc.SpanContext.TraceId)
	spanID := trace.SpanID{}
	copy(spanID[:], rc.SpanContext.SpanId)
	traceState, _ := trace.ParseTraceState(rc.SpanContext.TraceState)
	return trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.TraceFlags(rc.SpanContext.TraceFlags),
		TraceState: traceState,
		Remote:     rc.SpanContext.Remote,
	}))
}
