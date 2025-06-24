// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func fakeSpanContext(tb testing.TB) trace.SpanContext {
	traceID, err := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	require.NoError(tb, err)
	spanID, err := trace.SpanIDFromHex("0102030405060708")
	require.NoError(tb, err)
	traceState, err := trace.ParseTraceState("key1=value1,key2=value2,key3=value3")
	require.NoError(tb, err)
	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: 0x01,
		TraceState: traceState,
		Remote:     true,
	})
}
