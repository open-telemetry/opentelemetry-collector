// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/client"
	pdataint "go.opentelemetry.io/collector/pdata/internal"
	protocommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/xpdata/request/internal"
)

var readOnlyState = pdataint.StateReadOnly

// encodeContext encodes the context into a map of strings.
func encodeContext(ctx context.Context) internal.RequestContext {
	return internal.RequestContext{
		SpanContext:    encodeSpanContext(ctx),
		ClientMetadata: encodeClientMetadata(ctx),
	}
}

func encodeSpanContext(ctx context.Context) *internal.SpanContext {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return nil
	}
	traceID := spanCtx.TraceID()
	spanID := spanCtx.SpanID()
	return &internal.SpanContext{
		TraceId:    traceID[:],
		SpanId:     spanID[:],
		TraceFlags: uint32(spanCtx.TraceFlags()),
		TraceState: spanCtx.TraceState().String(),
		Remote:     spanCtx.IsRemote(),
	}
}

func encodeClientMetadata(ctx context.Context) []protocommon.KeyValue {
	clientMetadata := client.FromContext(ctx).Metadata
	metadataMap, metadataFound := pcommon.Map{}, false
	for k := range clientMetadata.Keys() {
		if !metadataFound {
			metadataMap, metadataFound = pcommon.NewMap(), true
		}
		vals := metadataMap.PutEmptySlice(k)
		for i := 0; i < len(clientMetadata.Get(k)); i++ {
			vals.AppendEmpty().SetStr(clientMetadata.Get(k)[i])
		}
	}
	if metadataFound {
		return *pdataint.GetOrigMap(pdataint.Map(metadataMap))
	}
	return nil
}

// decodeContext decodes the context from the bytes map.
func decodeContext(rc *internal.RequestContext) context.Context {
	ctx := context.Background()
	if rc == nil {
		return ctx
	}
	ctx = decodeSpanContext(ctx, rc.SpanContext)
	return decodeClientMetadata(ctx, rc.ClientMetadata)
}

func decodeSpanContext(ctx context.Context, sc *internal.SpanContext) context.Context {
	if sc == nil {
		return ctx
	}
	traceID := trace.TraceID{}
	copy(traceID[:], sc.TraceId)
	spanID := trace.SpanID{}
	copy(spanID[:], sc.SpanId)
	traceState, _ := trace.ParseTraceState(sc.TraceState)
	return trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.TraceFlags(sc.TraceFlags),
		TraceState: traceState,
		Remote:     sc.Remote,
	}))
}

func decodeClientMetadata(ctx context.Context, clientMetadata []protocommon.KeyValue) context.Context {
	if len(clientMetadata) == 0 {
		return ctx
	}
	metadataMap := make(map[string][]string, len(clientMetadata))
	for k, vals := range pcommon.Map(pdataint.NewMap(&clientMetadata, &readOnlyState)).All() {
		metadataMap[k] = make([]string, vals.Slice().Len())
		for i, v := range vals.Slice().All() {
			metadataMap[k][i] = v.Str()
		}
	}
	return client.NewContext(ctx, client.Info{Metadata: client.NewMetadata(metadataMap)})
}
