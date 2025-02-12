// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batcher

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component/componenttest"
)

func TestMergedContextDeadline(t *testing.T) {
	now := time.Now()
	ctx1 := context.Background()
	mergedContext := newMergedContext(ctx1)

	var ok bool
	deadline, ok := mergedContext.Deadline()
	require.Equal(t, time.Time{}, deadline)
	require.False(t, ok)

	ctx2, cancel2 := context.WithDeadline(context.Background(), now.Add(200))
	defer cancel2()
	mergedContext = mergedContext.Merge(ctx2)

	deadline, ok = mergedContext.Deadline()
	require.True(t, ok)
	require.Equal(t, now.Add(200), deadline)

	ctx3, cancel3 := context.WithDeadline(context.Background(), now.Add(300))
	defer cancel3()
	ctx4, cancel4 := context.WithDeadline(context.Background(), now.Add(100))
	defer cancel4()
	mergedContext = mergedContext.Merge(ctx3)
	mergedContext = mergedContext.Merge(ctx4)

	deadline, ok = mergedContext.Deadline()
	require.True(t, ok)
	require.Equal(t, now.Add(300), deadline)

	time.Sleep(300)
	require.Equal(t, ctx3.Err(), mergedContext.Err())
}

func TestMergedContextLink(t *testing.T) {
	tracerProvider := componenttest.NewTelemetry().NewTelemetrySettings().TracerProvider
	tracer := tracerProvider.Tracer("go.opentelemetry.io/collector/exporter/exporterhelper")

	ctx1 := context.Background()
	ctx2, span2 := tracer.Start(ctx1, "span2")
	defer span2.End()

	mergedContext := newMergedContext(ctx2)
	ctx3, span3 := tracer.Start(ctx1, "span3")
	defer span3.End()
	mergedContext = mergedContext.Merge(ctx3)
	ctx4, span4 := tracer.Start(ctx1, "span3")
	defer span4.End()
	mergedContext = mergedContext.Merge(ctx4)

	span2.AddEvent("This is an event.")

	spanFromMergedContext := trace.SpanFromContext(mergedContext)
	require.Equal(t, span2, spanFromMergedContext)

	require.Equal(t, trace.SpanContextFromContext(ctx3), span2.(sdktrace.ReadOnlySpan).Links()[0].SpanContext)
	require.Equal(t, trace.SpanContextFromContext(ctx4), span2.(sdktrace.ReadOnlySpan).Links()[1].SpanContext)
}
