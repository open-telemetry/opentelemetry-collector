// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptrace // import "go.opentelemetry.io/collector/pdata/ptrace"

import (
	"go.opentelemetry.io/collector/pdata/internal"
	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
)

var _ MarshalSizer = (*ProtoMarshaler)(nil)

type ProtoMarshaler struct{}

func (e *ProtoMarshaler) MarshalTraces(td Traces) ([]byte, error) {
	size := internal.SizeProtoOrigExportTraceServiceRequest(td.getOrig())
	buf := make([]byte, size)
	_ = internal.MarshalProtoOrigExportTraceServiceRequest(td.getOrig(), buf)
	return buf, nil
}

func (e *ProtoMarshaler) TracesSize(td Traces) int {
	return internal.SizeProtoOrigExportTraceServiceRequest(td.getOrig())
}

func (e *ProtoMarshaler) ResourceSpansSize(rs ResourceSpans) int {
	return internal.SizeProtoOrigResourceSpans(rs.orig)
}

func (e *ProtoMarshaler) ScopeSpansSize(ss ScopeSpans) int {
	return internal.SizeProtoOrigScopeSpans(ss.orig)
}

func (e *ProtoMarshaler) SpanSize(span Span) int {
	return internal.SizeProtoOrigSpan(span.orig)
}

type ProtoUnmarshaler struct{}

func (d *ProtoUnmarshaler) UnmarshalTraces(buf []byte) (Traces, error) {
	pb := otlptrace.TracesData{}
	err := pb.Unmarshal(buf)
	return Traces(internal.TracesFromProto(pb)), err
}
