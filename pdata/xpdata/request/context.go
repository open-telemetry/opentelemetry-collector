// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"
	"net"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/pdata/internal"
)

// encodeContext encodes the context into a map of strings.
func encodeContext(ctx context.Context) *internal.RequestContext {
	rc := internal.RequestContext{}
	encodeSpanContext(ctx, &rc)
	encodeClientMetadata(ctx, &rc)
	encodeClientAddress(ctx, &rc)
	return &rc
}

func encodeSpanContext(ctx context.Context, rc *internal.RequestContext) {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return
	}
	rc.SpanContext = &internal.SpanContext{
		TraceID:    internal.TraceID(spanCtx.TraceID()),
		SpanID:     internal.SpanID(spanCtx.SpanID()),
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
			rc.ClientMetadata = append(rc.ClientMetadata, internal.KeyValue{
				Key:   k,
				Value: internal.AnyValue{Value: &internal.AnyValue_StringValue{StringValue: vals[0]}},
			})
		default:
			metadataArray := make([]internal.AnyValue, 0, len(vals))
			for i := range vals {
				metadataArray = append(metadataArray, internal.AnyValue{Value: &internal.AnyValue_StringValue{StringValue: vals[i]}})
			}
			rc.ClientMetadata = append(rc.ClientMetadata, internal.KeyValue{
				Key:   k,
				Value: internal.AnyValue{Value: &internal.AnyValue_ArrayValue{ArrayValue: &internal.ArrayValue{Values: metadataArray}}},
			})
		}
	}
}

func encodeClientAddress(ctx context.Context, rc *internal.RequestContext) {
	switch a := client.FromContext(ctx).Addr.(type) {
	case *net.IPAddr:
		rc.ClientAddress = &internal.RequestContext_IP{IP: &internal.IPAddr{
			IP:   a.IP,
			Zone: a.Zone,
		}}
	case *net.TCPAddr:
		rc.ClientAddress = &internal.RequestContext_TCP{TCP: &internal.TCPAddr{
			IP:   a.IP,
			Port: int64(a.Port),
			Zone: a.Zone,
		}}
	case *net.UDPAddr:
		rc.ClientAddress = &internal.RequestContext_UDP{UDP: &internal.UDPAddr{
			IP:   a.IP,
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
	traceID := trace.TraceID(sc.TraceID)
	spanID := trace.SpanID(sc.SpanID)
	traceState, _ := trace.ParseTraceState(sc.TraceState)
	return trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.TraceFlags(sc.TraceFlags),
		TraceState: traceState,
		Remote:     sc.Remote,
	}))
}

func decodeClientMetadata(clientMetadata []internal.KeyValue) map[string][]string {
	if len(clientMetadata) == 0 {
		return nil
	}
	metadataMap := make(map[string][]string, len(clientMetadata))
	for _, kv := range clientMetadata {
		switch val := kv.Value.Value.(type) {
		case *internal.AnyValue_StringValue:
			metadataMap[kv.Key] = make([]string, 1)
			metadataMap[kv.Key][0] = val.StringValue
		case *internal.AnyValue_ArrayValue:
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
	case *internal.RequestContext_IP:
		return &net.IPAddr{
			IP:   a.IP.IP,
			Zone: a.IP.Zone,
		}
	case *internal.RequestContext_TCP:
		return &net.TCPAddr{
			IP:   a.TCP.IP,
			Port: int(a.TCP.Port),
			Zone: a.TCP.Zone,
		}
	case *internal.RequestContext_UDP:
		return &net.UDPAddr{
			IP:   a.UDP.IP,
			Port: int(a.UDP.Port),
			Zone: a.UDP.Zone,
		}
	case *internal.RequestContext_Unix:
		return &net.UnixAddr{
			Name: a.Unix.Name,
			Net:  a.Unix.Net,
		}
	}
	return nil
}
