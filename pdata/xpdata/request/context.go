// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"

	"go.opentelemetry.io/otel/propagation"

	pdataint "go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/xpdata/request/internal"
)

// Default trace context propagator
var tracePropagator = propagation.TraceContext{}

// encodeContext encodes the context into a map of strings.
func encodeContext(ctx context.Context) internal.RequestContext {
	carrier := pdataMapCarrier(pcommon.NewMap())
	tracePropagator.Inject(ctx, carrier)
	return internal.RequestContext{SpanContextMap: *pdataint.GetOrigMap(pdataint.Map(carrier))}
}

// decodeContext decodes the context from the bytes map.
func decodeContext(rc *internal.RequestContext) context.Context {
	if rc == nil || rc.SpanContextMap == nil {
		return context.Background()
	}
	state := pdataint.StateMutable
	carrier := pdataMapCarrier(pdataint.NewMap(&rc.SpanContextMap, &state))
	return tracePropagator.Extract(context.Background(), carrier)
}

type pdataMapCarrier pcommon.Map

var _ propagation.TextMapCarrier = pdataMapCarrier{}

func (m pdataMapCarrier) Get(key string) string {
	v, ok := pcommon.Map(m).Get(key)
	if !ok || v.Type() != pcommon.ValueTypeStr {
		return ""
	}
	return v.Str()
}

func (m pdataMapCarrier) Set(key, value string) {
	pcommon.Map(m).PutStr(key, value)
}

func (m pdataMapCarrier) Keys() []string {
	keys := make([]string, 0, pcommon.Map(m).Len())
	pcommon.Map(m).Range(func(k string, _ pcommon.Value) bool {
		keys = append(keys, k)
		return true
	})
	return keys
}
