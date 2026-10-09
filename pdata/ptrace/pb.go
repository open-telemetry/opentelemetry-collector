// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptrace // import "go.opentelemetry.io/collector/pdata/ptrace"

var _ MarshalSizer = (*ProtoMarshaler)(nil)

type ProtoMarshaler struct{}

func (e *ProtoMarshaler) MarshalTraces(td Traces) ([]byte, error) {
	size := td.getOrig().SizeProto()
	buf := make([]byte, size)
	_ = td.getOrig().MarshalProto(buf)
	return buf, nil
}

func (e *ProtoMarshaler) TracesSize(td Traces) int {
	return td.getOrig().SizeProto()
}

func (e *ProtoMarshaler) ResourceSpansSize(td ResourceSpans) int {
	return td.orig.SizeProto()
}

func (e *ProtoMarshaler) ScopeSpansSize(td ScopeSpans) int {
	return td.orig.SizeProto()
}

func (e *ProtoMarshaler) SpanSize(td Span) int {
	return td.orig.SizeProto()
}

type ProtoUnmarshaler struct{}

// UnmarshalTraces decodes buf into Traces.
//
// With the pdata.useProtoPooling feature gate enabled the result borrows its string and bytes
// fields straight from buf, so the caller must not modify or reuse buf while the result is in use.
func (d *ProtoUnmarshaler) UnmarshalTraces(buf []byte) (Traces, error) {
	td := NewTraces()
	td.getState().RetainWire(buf)
	err := td.getOrig().UnmarshalProtoState(buf, td.getState())
	if err != nil {
		return Traces{}, err
	}
	return td, nil
}
