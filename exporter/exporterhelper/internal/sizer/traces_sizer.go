// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sizer // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"

import (
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type TracesSizer interface {
	TracesSize(ld ptrace.Traces) int
	ResourceSpansSize(rs ptrace.ResourceSpans) int
	ScopeSpansSize(ss ptrace.ScopeSpans) int
	SpanSize(span ptrace.Span) int
	// DeltaSize() returns the delta size when a span is added.
	DeltaSize(newItemSize int) int
}

// TracesBytesSizer returns the byte size of serialized protos.
type TracesBytesSizer struct {
	ptrace.ProtoMarshaler
	protoDeltaSizer
}

// TracesCountSizer returns the number of spans in the traces.
type TracesCountSizer struct{}

func (s *TracesCountSizer) TracesSize(td ptrace.Traces) int {
	return td.SpanCount()
}

func (s *TracesCountSizer) ResourceSpansSize(rs ptrace.ResourceSpans) int {
	count := 0
	for k := 0; k < rs.ScopeSpans().Len(); k++ {
		count += rs.ScopeSpans().At(k).Spans().Len()
	}
	return count
}

func (s *TracesCountSizer) ScopeSpansSize(ss ptrace.ScopeSpans) int {
	return ss.Spans().Len()
}

func (s *TracesCountSizer) SpanSize(_ ptrace.Span) int {
	return 1
}

func (s *TracesCountSizer) DeltaSize(newItemSize int) int {
	return newItemSize
}
