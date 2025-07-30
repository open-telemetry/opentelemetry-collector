// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component/componenttest"
)

type testTimestampKeyType int

const testTimestampKey testTimestampKeyType = iota

// mergeCtxFunc corresponds to user specified mergeCtx function in the batcher settings.
// This specific merge Context function keeps the greater of timestamps from two contexts.
func mergeCtxFunc(ctx1, ctx2 context.Context) context.Context {
	timestamp1 := ctx1.Value(testTimestampKey)
	timestamp2 := ctx2.Value(testTimestampKey)
	if timestamp1 != nil && timestamp2 != nil {
		if timestamp1.(int) > timestamp2.(int) {
			return context.WithValue(context.Background(), testTimestampKey, timestamp1)
		}
		return context.WithValue(context.Background(), testTimestampKey, timestamp2)
	}
	if timestamp1 != nil {
		return context.WithValue(context.Background(), testTimestampKey, timestamp1)
	}
	return context.WithValue(context.Background(), testTimestampKey, timestamp2)
}

// mergeContextHelper performs the same operation done during batching.
func mergeContextHelper(ctx1, ctx2 context.Context) context.Context {
	return contextWithMergedLinks(mergeCtxFunc(ctx1, ctx2), ctx1, ctx2)
}

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

	batchContext := mergeContextHelper(ctx2, ctx3)
	batchContext = mergeContextHelper(batchContext, ctx4)

	actualLinks := LinksFromContext(batchContext)
	require.Len(t, actualLinks, 3)
	require.Equal(t, trace.SpanContextFromContext(ctx2), actualLinks[0].SpanContext)
	require.Equal(t, trace.SpanContextFromContext(ctx3), actualLinks[1].SpanContext)
	require.Equal(t, trace.SpanContextFromContext(ctx4), actualLinks[2].SpanContext)
}

func TestMergedContext_GetValue(t *testing.T) {
	ctx1 := context.WithValue(context.Background(), testTimestampKey, 1234)
	ctx2 := context.WithValue(context.Background(), testTimestampKey, 2345)
	batchContext := mergeContextHelper(ctx1, ctx2)
	require.Equal(t, 2345, batchContext.Value(testTimestampKey))
}
