// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"

	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/collector/pdata/xpdata/request/internal"
)

// Default trace context propagator
var tracePropagator = propagation.TraceContext{}

// Trace propagator sets only two keys to encode the context: "traceparent" and "tracestate"
const reqContextKeysNum = 2

// encodeContext encodes the context into a map of strings.
func encodeContext(ctx context.Context) internal.RequestContext {
	mc := propagation.MapCarrier(make(map[string]string, reqContextKeysNum))
	tracePropagator.Inject(ctx, mc)
	return internal.RequestContext{SpanContextMap: mc}
}

// decodeContext decodes the context from the bytes map.
func decodeContext(rc *internal.RequestContext) context.Context {
	if rc == nil || rc.SpanContextMap == nil {
		return context.Background()
	}
	return tracePropagator.Extract(context.Background(), propagation.MapCarrier(rc.SpanContextMap))
}
