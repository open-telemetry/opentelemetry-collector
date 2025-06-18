// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/pdata/xpdata/request/internal"
)

func TestEncodeDecodeContext(t *testing.T) {
	spanCtx := fakeSpanContext(t)
	clientMetadata := client.NewMetadata(map[string][]string{
		"key1": {"value1"},
		"key2": {"value2", "value3"},
	})

	// Encode a context with a span and client metadata
	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)
	ctx = client.NewContext(ctx, client.Info{
		Metadata: clientMetadata,
	})
	reqCtx := encodeContext(ctx)
	buf, err := reqCtx.Marshal()
	require.NoError(t, err)

	// Decode the context
	gotReqCtx := internal.RequestContext{}
	err = gotReqCtx.Unmarshal(buf)
	require.NoError(t, err)
	gotCtx := decodeContext(context.Background(), &gotReqCtx)
	assert.Equal(t, spanCtx, trace.SpanContextFromContext(gotCtx))
	assert.Equal(t, clientMetadata, client.FromContext(gotCtx).Metadata)

	// Decode nil request context
	assert.Equal(t, context.Background(), decodeContext(context.Background(), nil))
}
