// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptext // import "go.opentelemetry.io/collector/exporter/internal/otlptext"

import (
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// NewTextTracesMarshaler returns a ptrace.Marshaler to encode to OTLP text bytes.
func NewTextTracesMarshaler() ptrace.Marshaler {
	return textTracesMarshaler{}
}

type textTracesMarshaler struct{}

// MarshalTraces ptrace.Traces to OTLP text.
func (textTracesMarshaler) MarshalTraces(td ptrace.Traces) ([]byte, error) {
	buf := dataBuffer{}
	td.ResourceSpans().ForEachIndex(func(i int, rs ptrace.ResourceSpans) {
		buf.logEntry("ResourceSpans #%d", i)
		buf.logEntry("Resource SchemaURL: %s", rs.SchemaUrl())
		buf.logAttributes("Resource attributes", rs.Resource().Attributes())
		rs.ScopeSpans().ForEachIndex(func(j int, ils ptrace.ScopeSpans) {
			buf.logEntry("ScopeSpans #%d", j)
			buf.logEntry("ScopeSpans SchemaURL: %s", ils.SchemaUrl())
			buf.logInstrumentationScope(ils.Scope())
			ils.Spans().ForEachIndex(func(k int, span ptrace.Span) {
				buf.logEntry("Span #%d", k)
				buf.logAttr("Trace ID", span.TraceID())
				buf.logAttr("Parent ID", span.ParentSpanID())
				buf.logAttr("ID", span.SpanID())
				buf.logAttr("Name", span.Name())
				buf.logAttr("Kind", span.Kind().String())
				buf.logAttr("Start time", span.StartTimestamp().String())
				buf.logAttr("End time", span.EndTimestamp().String())

				buf.logAttr("Status code", span.Status().Code().String())
				buf.logAttr("Status message", span.Status().Message())

				buf.logAttributes("Attributes", span.Attributes())
				buf.logEvents("Events", span.Events())
				buf.logLinks("Links", span.Links())
			})
		})
	})
	return buf.buf.Bytes(), nil
}
