// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"testing"
	"time"

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

func TestContextWithMergedDeadline_BothHaveDeadlines(t *testing.T) {
	now := time.Now()
	d1 := now.Add(5 * time.Second)
	d2 := now.Add(10 * time.Second)

	ctx1, cancel1 := context.WithDeadline(context.Background(), d1)
	defer cancel1()
	ctx2, cancel2 := context.WithDeadline(context.Background(), d2)
	defer cancel2()

	merged := contextWithMergedDeadline(context.Background(), ctx1, ctx2)
	deadline, ok := deadlineFromContext(merged)
	require.True(t, ok)
	require.Equal(t, d2, deadline) // max of the two
}

func TestContextWithMergedDeadline_OnlyFirstHasDeadline(t *testing.T) {
	now := time.Now()
	d1 := now.Add(5 * time.Second)

	ctx1, cancel1 := context.WithDeadline(context.Background(), d1)
	defer cancel1()
	ctx2 := context.Background()

	merged := contextWithMergedDeadline(context.Background(), ctx1, ctx2)
	deadline, ok := deadlineFromContext(merged)
	require.True(t, ok)
	require.Equal(t, d1, deadline)
}

func TestContextWithMergedDeadline_OnlySecondHasDeadline(t *testing.T) {
	now := time.Now()
	d2 := now.Add(10 * time.Second)

	ctx1 := context.Background()
	ctx2, cancel2 := context.WithDeadline(context.Background(), d2)
	defer cancel2()

	merged := contextWithMergedDeadline(context.Background(), ctx1, ctx2)
	deadline, ok := deadlineFromContext(merged)
	require.True(t, ok)
	require.Equal(t, d2, deadline)
}

func TestContextWithMergedDeadline_NeitherHasDeadline(t *testing.T) {
	ctx1 := context.Background()
	ctx2 := context.Background()

	merged := contextWithMergedDeadline(context.Background(), ctx1, ctx2)
	_, ok := deadlineFromContext(merged)
	require.False(t, ok)
}

func TestContextWithMergedDeadline_AccumulatedAcrossMultipleMerges(t *testing.T) {
	now := time.Now()
	d1 := now.Add(5 * time.Second)
	d2 := now.Add(3 * time.Second)
	d3 := now.Add(10 * time.Second)

	ctx1, cancel1 := context.WithDeadline(context.Background(), d1)
	defer cancel1()
	ctx2, cancel2 := context.WithDeadline(context.Background(), d2)
	defer cancel2()
	ctx3, cancel3 := context.WithDeadline(context.Background(), d3)
	defer cancel3()

	// First merge: ctx1 + ctx2 → max is d1 (5s > 3s)
	merged := contextWithMergedDeadline(context.Background(), ctx1, ctx2)
	deadline, ok := deadlineFromContext(merged)
	require.True(t, ok)
	require.Equal(t, d1, deadline)

	// Second merge: merged (stored d1) + ctx3 → max is d3 (10s > 5s)
	merged = contextWithMergedDeadline(context.Background(), merged, ctx3)
	deadline, ok = deadlineFromContext(merged)
	require.True(t, ok)
	require.Equal(t, d3, deadline)
}
