// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/featuregate"
)

const (
	errInvalidTraceFlagsLength = "trace flags must only be 1 byte"
)

// persistRequestContextFeatureGate controls whether request context should be persisted in the queue.
var persistRequestContextFeatureGate = featuregate.GlobalRegistry().MustRegister(
	"exporter.PersistRequestContext",
	featuregate.StageAlpha,
	featuregate.WithRegisterFromVersion("v0.127.0"),
	featuregate.WithRegisterDescription("controls whether context should be stored alongside requests in the persistent queue"),
	featuregate.WithRegisterReferenceURL("https://github.com/open-telemetry/opentelemetry-collector/pull/12934"),
)

// necessary due to SpanContext and SpanContextConfig not supporting Unmarshal interface,
// see https://github.com/open-telemetry/opentelemetry-go/issues/1819.
type spanContext struct {
	TraceID    string
	SpanID     string
	TraceFlags string
	TraceState string
	Remote     bool
}

func localSpanContextFromTraceSpanContext(sc trace.SpanContext) spanContext {
	return spanContext{
		TraceID:    sc.TraceID().String(),
		SpanID:     sc.SpanID().String(),
		TraceFlags: sc.TraceFlags().String(),
		TraceState: sc.TraceState().String(),
		Remote:     sc.IsRemote(),
	}
}

func contextWithLocalSpanContext(ctx context.Context, sc spanContext) context.Context {
	traceID, err := trace.TraceIDFromHex(sc.TraceID)
	if err != nil {
		return ctx
	}
	spanID, err := trace.SpanIDFromHex(sc.SpanID)
	if err != nil {
		return ctx
	}
	traceFlags, err := traceFlagsFromHex(sc.TraceFlags)
	if err != nil {
		return ctx
	}
	traceState, err := trace.ParseTraceState(sc.TraceState)
	if err != nil {
		return ctx
	}

	return trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: *traceFlags,
		TraceState: traceState,
		Remote:     sc.Remote,
	}))
}

// requestContext wraps trace.SpanContext to allow for unmarshaling as well as
// future metadata key/value pairs to be added.
type requestContext struct {
	SpanContext spanContext
}

// reverse of code in trace library https://github.com/open-telemetry/opentelemetry-go/blob/v1.35.0/trace/trace.go#L143-L168
func traceFlagsFromHex(hexStr string) (*trace.TraceFlags, error) {
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, err
	}
	if len(decoded) != 1 {
		return nil, errors.New(errInvalidTraceFlagsLength)
	}
	traceFlags := trace.TraceFlags(decoded[0])
	return &traceFlags, nil
}

func getAndMarshalSpanContext(ctx context.Context) ([]byte, error) {
	if !persistRequestContextFeatureGate.IsEnabled() {
		return nil, nil
	}
	rc := localSpanContextFromTraceSpanContext(trace.SpanContextFromContext(ctx))
	return json.Marshal(requestContext{SpanContext: rc})
}

func getContextKey(index uint64) string {
	return strconv.FormatUint(index, 10) + "_context"
}
