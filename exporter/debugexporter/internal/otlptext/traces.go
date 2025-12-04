// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptext // import "go.opentelemetry.io/collector/exporter/debugexporter/internal/otlptext"

import (
	"strconv"

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
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		buf.logEntry("ResourceSpans #%d", i)
		rs := rss.At(i)
		buf.logEntry("Resource SchemaURL: %s", rs.SchemaUrl())
		buf.logAttributes("Resource attributes", rs.Resource().Attributes())
		buf.logEntityRefs(rs.Resource())
		ilss := rs.ScopeSpans()
		for j := 0; j < ilss.Len(); j++ {
			buf.logEntry("ScopeSpans #%d", j)
			ils := ilss.At(j)
			buf.logEntry("ScopeSpans SchemaURL: %s", ils.SchemaUrl())
			buf.logInstrumentationScope(ils.Scope())

			spans := ils.Spans()
			for k := 0; k < spans.Len(); k++ {
				buf.logEntry("Span #%d", k)
				span := spans.At(k)
				buf.logAttr("Trace ID", span.TraceID())
				buf.logAttr("Parent ID", span.ParentSpanID())
				buf.logAttr("ID", span.SpanID())
				buf.logAttr("Name", span.Name())
				buf.logAttr("Kind", span.Kind().String())
				if ts := span.TraceState().AsRaw(); ts != "" {
					buf.logAttr("TraceState", ts)
				}
				buf.logAttr("Start time", span.StartTimestamp().String())
				buf.logAttr("End time", span.EndTimestamp().String())

				buf.logAttr("Status code", span.Status().Code().String())
				buf.logAttr("Status message", span.Status().Message())

				buf.logAttr("DroppedAttributesCount", strconv.FormatUint(uint64(span.DroppedAttributesCount()), 10))
				buf.logAttr("DroppedEventsCount", strconv.FormatUint(uint64(span.DroppedEventsCount()), 10))
				buf.logAttr("DroppedLinksCount", strconv.FormatUint(uint64(span.DroppedLinksCount()), 10))

				buf.logAttributes("Attributes", span.Attributes())
				buf.logEvents("Events", span.Events())
				buf.logLinks("Links", span.Links())
			}
		}
	}

	return buf.buf.Bytes(), nil
}
