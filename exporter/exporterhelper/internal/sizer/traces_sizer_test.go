// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package sizer

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestTracesCountSizer(t *testing.T) {
	td := testdata.GenerateTraces(5)
	sizer := TracesCountSizer{}
	require.Equal(t, 5, sizer.TracesSize(td))

	rs := td.ResourceSpans().At(0)
	require.Equal(t, 5, sizer.ResourceSpansSize(rs))

	ss := rs.ScopeSpans().At(0)
	require.Equal(t, 5, sizer.ScopeSpansSize(ss))

	require.Equal(t, 1, sizer.SpanSize(ss.Spans().At(0)))
	require.Equal(t, 1, sizer.SpanSize(ss.Spans().At(1)))
	require.Equal(t, 1, sizer.SpanSize(ss.Spans().At(2)))
	require.Equal(t, 1, sizer.SpanSize(ss.Spans().At(3)))
	require.Equal(t, 1, sizer.SpanSize(ss.Spans().At(4)))

	prevSize := sizer.ScopeSpansSize(ss)
	span := ss.Spans().At(2)
	span.CopyTo(ss.Spans().AppendEmpty())
	require.Equal(t, sizer.ScopeSpansSize(ss), prevSize+sizer.DeltaSize(sizer.SpanSize(span)))
}

func TestTracesBytesSizer(t *testing.T) {
	td := testdata.GenerateTraces(2)
	sizer := TracesBytesSizer{}
	require.Equal(t, 338, sizer.TracesSize(td))

	rs := td.ResourceSpans().At(0)
	require.Equal(t, 335, sizer.ResourceSpansSize(rs))

	ss := rs.ScopeSpans().At(0)
	require.Equal(t, 290, sizer.ScopeSpansSize(ss))

	require.Equal(t, 187, sizer.SpanSize(ss.Spans().At(0)))
	require.Equal(t, 96, sizer.SpanSize(ss.Spans().At(1)))

	prevSize := sizer.ScopeSpansSize(ss)
	span := ss.Spans().At(1)
	spanSize := sizer.SpanSize(span)
	span.CopyTo(ss.Spans().AppendEmpty())
	ds := sizer.DeltaSize(spanSize)
	require.Equal(t, prevSize+ds, sizer.ScopeSpansSize(ss))
}
