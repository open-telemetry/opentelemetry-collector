// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptrace // import "go.opentelemetry.io/collector/pdata/ptrace"

import (
	"go.opentelemetry.io/collector/pdata/internal"
)

var _ MarshalSizer = (*ProtoMarshaler)(nil)

type ProtoMarshaler struct{}

func (e *ProtoMarshaler) MarshalTraces(td Traces) ([]byte, error) {
	size := internal.SizeProtoExportTraceServiceRequest(td.getOrig())
	buf := make([]byte, size)
	_ = internal.MarshalProtoExportTraceServiceRequest(td.getOrig(), buf)
	return buf, nil
}

func (e *ProtoMarshaler) TracesSize(td Traces) int {
	return internal.SizeProtoExportTraceServiceRequest(td.getOrig())
}

func (e *ProtoMarshaler) ResourceSpansSize(td ResourceSpans) int {
	return internal.SizeProtoResourceSpans(td.orig)
}

func (e *ProtoMarshaler) ScopeSpansSize(td ScopeSpans) int {
	return internal.SizeProtoScopeSpans(td.orig)
}

func (e *ProtoMarshaler) SpanSize(td Span) int {
	return internal.SizeProtoSpan(td.orig)
}

type ProtoUnmarshaler struct{}

func (d *ProtoUnmarshaler) UnmarshalTraces(buf []byte) (Traces, error) {
	td := NewTraces()
	err := internal.UnmarshalProtoExportTraceServiceRequest(td.getOrig(), buf)
	if err != nil {
		return Traces{}, err
	}
	return td, nil
}
