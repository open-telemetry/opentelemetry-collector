// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMarshalUnmarshalLogsRequest(t *testing.T) {
	logs := testdata.GenerateLogs(3)

	// unmarshal logs request with a context
	spanCtx := fakeSpanContext(t)
	buf, err := MarshalLogs(trace.ContextWithSpanContext(context.Background(), spanCtx), logs)
	require.NoError(t, err)
	gotCtx, gotLogs, err := UnmarshalLogs(buf)
	require.NoError(t, err)
	assert.Equal(t, spanCtx, trace.SpanContextFromContext(gotCtx))
	assert.Equal(t, logs, gotLogs)

	// unmarshal logs request with empty context
	buf, err = MarshalLogs(context.Background(), logs)
	require.NoError(t, err)
	gotCtx, gotLogs, err = UnmarshalLogs(buf)
	require.NoError(t, err)
	assert.Equal(t, context.Background(), gotCtx)
	assert.Equal(t, logs, gotLogs)

	// unmarshal corrupted data
	_, _, err = UnmarshalLogs(buf[:len(buf)-1])
	require.ErrorContains(t, err, "failed to unmarshal logs request")

	// unmarshal invalid format (bare logs)
	buf, err = (&plog.ProtoMarshaler{}).MarshalLogs(logs)
	require.NoError(t, err)
	_, _, err = UnmarshalLogs(buf)
	require.ErrorIs(t, err, ErrInvalidFormat)
}
