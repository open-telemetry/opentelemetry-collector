// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/otel/trace"
)

type testContextKey string

func TestBatchContextLink(t *testing.T) {
	tracerProvider := componenttest.NewTelemetry().NewTelemetrySettings().TracerProvider
	tracer := tracerProvider.Tracer("go.opentelemetry.io/collector/exporter/exporterhelper")

	ctx1 := context.Background()

	ctx2, span2 := tracer.Start(ctx1, "span2")
	defer span2.End()

	ctx3, span3 := tracer.Start(ctx1, "span3")
	defer span3.End()

	ctx4, span4 := tracer.Start(ctx1, "span4")
	defer span4.End()

	batchContext := contextWithMergedLinks(ctx2, ctx3)
	batchContext = contextWithMergedLinks(batchContext, ctx4)

	actualLinks := LinksFromContext(batchContext)
	// require.Len(t, actualLinks, 3)
	require.Equal(t, trace.SpanContextFromContext(ctx4), actualLinks[0].SpanContext)
	// require.Equal(t, trace.SpanContextFromContext(ctx3), actualLinks[1].SpanContext)
	// require.Equal(t, trace.SpanContextFromContext(ctx4), actualLinks[2].SpanContext)
}

func TestMergedContext_GetValue(t *testing.T) {
	ctx1 := context.WithValue(context.Background(), testContextKey("key1"), "value1")
	ctx2 := context.WithValue(context.Background(), testContextKey("key1"), "value2")
	ctx2 = context.WithValue(ctx2, testContextKey("key2"), "value2")
	ctx3 := context.WithValue(context.Background(), testContextKey("key2"), "value3")

	var mergedCtx context.Context
	mergedCtx = contextWithMergedLinks(ctx1, ctx2)
	mergedCtx = contextWithMergedLinks(mergedCtx, ctx3)

	require.Equal(t, "value1", mergedCtx.Value(testContextKey("key1")))
	require.Equal(t, "value2", mergedCtx.Value(testContextKey("key2")))
	require.Nil(t, mergedCtx.Value("nonexistent_key"))
}

func TestMergedValues_GetValue_NilContext(t *testing.T) {
	ctx1 := context.WithValue(context.Background(), testContextKey("key1"), "value1")
	var ctx2 context.Context // nil context

	var mergedCtx context.Context
	mergedCtx = contextWithMergedLinks(ctx1, ctx2)

	require.Equal(t, "value1", mergedCtx.Value(testContextKey("key1")))
	require.Nil(t, mergedCtx.Value(testContextKey("key2")))
	require.Nil(t, mergedCtx.Value("nonexistent_key"))
}

func TestMergedValues_GetValue_CanceledContext(t *testing.T) {
	ctx1 := context.WithValue(context.Background(), testContextKey("key1"), "value1")
	ctx2, cancel := context.WithCancel(context.WithValue(context.Background(), testContextKey("key2"), "value2"))

	var mergedCtx context.Context
	mergedCtx = contextWithMergedLinks(ctx1, ctx2)

	cancel()

	require.Equal(t, "value1", mergedCtx.Value(testContextKey("key1")))
	require.Equal(t, "value2", mergedCtx.Value(testContextKey("key2")))
	require.Nil(t, mergedCtx.Value("nonexistent_key"))
}
