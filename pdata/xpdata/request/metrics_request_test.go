// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMarshalUnmarshalMetricsRequest(t *testing.T) {
	metrics := testdata.GenerateMetrics(3)

	// unmarshal metrics request with a context
	spanCtx := fakeSpanContext(t)
	buf, err := MarshalMetrics(trace.ContextWithSpanContext(context.Background(), spanCtx), metrics)
	require.NoError(t, err)
	gotCtx, gotMetrics, err := UnmarshalMetrics(buf)
	require.NoError(t, err)
	assert.Equal(t, spanCtx, trace.SpanContextFromContext(gotCtx))
	assert.Equal(t, metrics, gotMetrics)

	// unmarshal metrics request with empty context
	buf, err = MarshalMetrics(context.Background(), metrics)
	require.NoError(t, err)
	gotCtx, gotMetrics, err = UnmarshalMetrics(buf)
	require.NoError(t, err)
	assert.Equal(t, context.Background(), gotCtx)
	assert.Equal(t, metrics, gotMetrics)

	// unmarshal corrupted data
	_, _, err = UnmarshalMetrics(buf[:len(buf)-1])
	require.ErrorContains(t, err, "failed to unmarshal metrics request")

	// unmarshal invalid format (bare metrics)
	buf, err = (&pmetric.ProtoMarshaler{}).MarshalMetrics(metrics)
	require.NoError(t, err)
	_, _, err = UnmarshalMetrics(buf)
	require.ErrorIs(t, err, ErrInvalidFormat)
}
