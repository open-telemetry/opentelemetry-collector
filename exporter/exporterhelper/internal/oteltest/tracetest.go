// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package oteltest // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/oteltest"

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func CheckStatus(t *testing.T, sd sdktrace.ReadOnlySpan, err error) {
	if err != nil {
		require.Equal(t, codes.Error, sd.Status().Code)
		require.EqualError(t, err, sd.Status().Description)
	} else {
		require.Equal(t, codes.Unset, sd.Status().Code)
	}
}

func FakeSpanContext(t *testing.T) trace.SpanContext {
	traceID, err := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	require.NoError(t, err)
	spanID, err := trace.SpanIDFromHex("0102030405060708")
	require.NoError(t, err)
	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: 0x01,
		TraceState: trace.TraceState{},
		Remote:     true,
	})
}
