// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"
	"net"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/client"
	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
	"go.opentelemetry.io/collector/pdata/xpdata/request/internal"
)

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
	for k := range clientMetadata.Keys() {
		vals := clientMetadata.Get(k)
		switch len(vals) {
		case 1:
			rc.ClientMetadata = append(rc.ClientMetadata, otlpcommon.KeyValue{
				Key:   k,
				Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: vals[0]}},
			})
		default:
			metadataArray := make([]otlpcommon.AnyValue, 0, len(vals))
			for i := range vals {
				metadataArray = append(metadataArray, otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: vals[i]}})
			}
			rc.ClientMetadata = append(rc.ClientMetadata, otlpcommon.KeyValue{
				Key:   k,
				Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_ArrayValue{ArrayValue: &otlpcommon.ArrayValue{Values: metadataArray}}},
			})
		}
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

func decodeClientMetadata(clientMetadata []otlpcommon.KeyValue) map[string][]string {
	if len(clientMetadata) == 0 {
		return nil
	}
	metadataMap := make(map[string][]string, len(clientMetadata))
	for _, kv := range clientMetadata {
		switch val := kv.Value.Value.(type) {
		case *otlpcommon.AnyValue_StringValue:
			metadataMap[kv.Key] = make([]string, 1)
			metadataMap[kv.Key][0] = val.StringValue
		case *otlpcommon.AnyValue_ArrayValue:
			metadataMap[kv.Key] = make([]string, len(val.ArrayValue.Values))
			for i, v := range val.ArrayValue.Values {
				metadataMap[kv.Key][i] = v.GetStringValue()
			}
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
