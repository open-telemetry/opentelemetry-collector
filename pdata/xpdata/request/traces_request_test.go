// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMarshalUnmarshalTracesRequest(t *testing.T) {
	traces := testdata.GenerateTraces(3)

	// unmarshal traces request with a context
	spanCtx := fakeSpanContext(t)
	buf, err := MarshalTraces(trace.ContextWithSpanContext(context.Background(), spanCtx), traces)
	require.NoError(t, err)
	gotCtx, gotTraces, err := UnmarshalTraces(buf)
	require.NoError(t, err)
	assert.Equal(t, spanCtx, trace.SpanContextFromContext(gotCtx))
	assert.Equal(t, traces, gotTraces)

	// unmarshal traces request with empty context
	buf, err = MarshalTraces(context.Background(), traces)
	require.NoError(t, err)
	gotCtx, gotTraces, err = UnmarshalTraces(buf)
	require.NoError(t, err)
	assert.Equal(t, context.Background(), gotCtx)
	assert.Equal(t, traces, gotTraces)

	// unmarshal corrupted data
	_, _, err = UnmarshalTraces(buf[:len(buf)-1])
	require.ErrorContains(t, err, "failed to unmarshal traces request")

	// unmarshal invalid format (bare traces)
	buf, err = (&ptrace.ProtoMarshaler{}).MarshalTraces(traces)
	require.NoError(t, err)
	_, _, err = UnmarshalTraces(buf)
	require.ErrorIs(t, err, ErrInvalidFormat)
}
