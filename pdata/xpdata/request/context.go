// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"
	"net"

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
	rc := internal.RequestContext{}
	encodeSpanContext(ctx, &rc)
	encodeClientMetadata(ctx, &rc)
	encodeClientAddress(ctx, &rc)
	return rc
}

func encodeSpanContext(ctx context.Context, rc *internal.RequestContext) {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return
	}
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

func encodeClientMetadata(ctx context.Context, rc *internal.RequestContext) {
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
		rc.ClientMetadata = *pdataint.GetOrigMap(pdataint.Map(metadataMap))
	}
}

func encodeClientAddress(ctx context.Context, rc *internal.RequestContext) {
	switch a := client.FromContext(ctx).Addr.(type) {
	case *net.IPAddr:
		rc.ClientAddress = &internal.RequestContext_Ip{Ip: &internal.IPAddr{
			Ip:   a.IP,
			Zone: a.Zone,
		}}
	case *net.TCPAddr:
		rc.ClientAddress = &internal.RequestContext_Tcp{Tcp: &internal.TCPAddr{
			Ip:   a.IP,
			Port: int64(a.Port),
			Zone: a.Zone,
		}}
	case *net.UDPAddr:
		rc.ClientAddress = &internal.RequestContext_Udp{Udp: &internal.UDPAddr{
			Ip:   a.IP,
			Port: int64(a.Port),
			Zone: a.Zone,
		}}
	case *net.UnixAddr:
		rc.ClientAddress = &internal.RequestContext_Unix{Unix: &internal.UnixAddr{
			Name: a.Name,
			Net:  a.Net,
		}}
	}
}

// decodeContext decodes the context from the bytes map.
func decodeContext(ctx context.Context, rc *internal.RequestContext) context.Context {
	if rc == nil {
		return ctx
	}
	ctx = decodeSpanContext(ctx, rc.SpanContext)
	metadataMap := decodeClientMetadata(rc.ClientMetadata)
	clientAddress := decodeClientAddress(rc)
	if len(metadataMap) > 0 || clientAddress != nil {
		ctx = client.NewContext(ctx, client.Info{
			Metadata: client.NewMetadata(metadataMap),
			Addr:     clientAddress,
		})
	}
	return ctx
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

func decodeClientMetadata(clientMetadata []protocommon.KeyValue) map[string][]string {
	if len(clientMetadata) == 0 {
		return nil
	}
	metadataMap := make(map[string][]string, len(clientMetadata))
	for k, vals := range pcommon.Map(pdataint.NewMap(&clientMetadata, &readOnlyState)).All() {
		metadataMap[k] = make([]string, vals.Slice().Len())
		for i, v := range vals.Slice().All() {
			metadataMap[k][i] = v.Str()
		}
	}
	return metadataMap
}

func decodeClientAddress(rc *internal.RequestContext) net.Addr {
	switch a := rc.ClientAddress.(type) {
	case *internal.RequestContext_Ip:
		return &net.IPAddr{
			IP:   a.Ip.Ip,
			Zone: a.Ip.Zone,
		}
	case *internal.RequestContext_Tcp:
		return &net.TCPAddr{
			IP:   a.Tcp.Ip,
			Port: int(a.Tcp.Port),
			Zone: a.Tcp.Zone,
		}
	case *internal.RequestContext_Udp:
		return &net.UDPAddr{
			IP:   a.Udp.Ip,
			Port: int(a.Udp.Port),
			Zone: a.Udp.Zone,
		}
	case *internal.RequestContext_Unix:
		return &net.UnixAddr{
			Name: a.Unix.Name,
			Net:  a.Unix.Net,
		}
	}
	return nil
}
