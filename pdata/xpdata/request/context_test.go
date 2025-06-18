// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/pdata/xpdata/request/internal"
)

func BenchmarkEncodeDecodeContext(b *testing.B) {
	spanCtx := fakeSpanContext(b)
	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		reqCtx := encodeContext(ctx)
		buf, err := reqCtx.Marshal()
		require.NoError(b, err)

		gotReqCtx := internal.RequestContext{}
		err = gotReqCtx.Unmarshal(buf)
		require.NoError(b, err)
		gotCtx := decodeContext(&gotReqCtx)
		require.Equal(b, spanCtx, trace.SpanContextFromContext(gotCtx))
	}
}
