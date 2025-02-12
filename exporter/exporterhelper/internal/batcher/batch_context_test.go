// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batcher

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/otel/trace"
)

func TestBatchContextDeadline(t *testing.T) {
	now := time.Now()
	ctx1 := context.Background()
	batchContext := newBatchContext(ctx1)

	var ok bool
	deadline, ok := batchContext.Deadline()
	require.Equal(t, time.Time{}, deadline)
	require.False(t, ok)

	ctx2, cancel2 := context.WithDeadline(context.Background(), now.Add(200))
	defer cancel2()
	batchContext = batchContext.Merge(ctx2)

	deadline, ok = batchContext.Deadline()
	require.True(t, ok)
	require.Equal(t, now.Add(200), deadline)

	ctx3, cancel3 := context.WithDeadline(context.Background(), now.Add(300))
	defer cancel3()
	ctx4, cancel4 := context.WithDeadline(context.Background(), now.Add(100))
	defer cancel4()
	batchContext = batchContext.Merge(ctx3)
	batchContext = batchContext.Merge(ctx4)

	deadline, ok = batchContext.Deadline()
	require.True(t, ok)
	require.Equal(t, now.Add(300), deadline)

	time.Sleep(300)
	require.Equal(t, ctx3.Err(), batchContext.Err())
}

func TestBatchContextLink(t *testing.T) {
	tracerProvider := componenttest.NewTelemetry().NewTelemetrySettings().TracerProvider
	tracer := tracerProvider.Tracer("go.opentelemetry.io/collector/exporter/exporterhelper")

	ctx1 := context.Background()

	ctx2, span2 := tracer.Start(ctx1, "span2")
	defer span2.End()
	batchContext := newBatchContext(ctx2)

	ctx3, span3 := tracer.Start(ctx1, "span3")
	defer span3.End()
	batchContext = batchContext.Merge(ctx3)

	ctx4, span4 := tracer.Start(ctx1, "span4")
	defer span4.End()
	batchContext = batchContext.Merge(ctx4)

	span2.AddEvent("This is an event.")

	actualLinks := SpanLinksFromContext(batchContext)
	require.Len(t, actualLinks, 3)
	require.Equal(t, trace.SpanContextFromContext(ctx2), actualLinks[0].SpanContext)
	require.Equal(t, trace.SpanContextFromContext(ctx3), actualLinks[1].SpanContext)
	require.Equal(t, trace.SpanContextFromContext(ctx4), actualLinks[2].SpanContext)
}
