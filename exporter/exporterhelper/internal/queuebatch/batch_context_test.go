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
	require.Len(t, actualLinks, 3)
	require.Equal(t, trace.SpanContextFromContext(ctx2), actualLinks[0].SpanContext)
	require.Equal(t, trace.SpanContextFromContext(ctx3), actualLinks[1].SpanContext)
	require.Equal(t, trace.SpanContextFromContext(ctx4), actualLinks[2].SpanContext)
}
