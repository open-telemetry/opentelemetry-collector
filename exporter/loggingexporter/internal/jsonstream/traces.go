// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jsonstream

import (
	"go.opentelemetry.io/collector/model/pdata"
)

// NewJSONTracesMarshaler returns a serializer.TracesMarshaler to encode to OTLP JSON bytes.
func NewJSONTracesMarshaler() pdata.TracesMarshaler {
	return jsonTracesMarshaler{}
}

type jsonTracesMarshaler struct{}

// MarshalTraces pdata.Traces to OTLP JSON.
func (jsonTracesMarshaler) MarshalTraces(td pdata.Traces) ([]byte, error) {
	buf := dataBuffer{}
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)

		attrs := rs.Resource().Attributes()

		ilss := rs.InstrumentationLibrarySpans()
		if ilss.Len() == 0 {
			buf.object(func() {
				buf.resource("resourceSpan", attrs)
			})
			continue
		}
		for j := 0; j < ilss.Len(); j++ {
			ils := ilss.At(j)

			lib := ils.InstrumentationLibrary()

			spans := ils.Spans()
			if spans.Len() == 0 {
				buf.object(func() {
					buf.resource("instrumentationLibrarySpan", attrs)
					buf.instrumentationLibrary(lib)
				})
				continue
			}
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)

				buf.object(func() {
					buf.resource("span", attrs)
					buf.instrumentationLibrary(lib)

					buf.fieldString("traceID", span.TraceID().HexString())
					buf.fieldString("parentID", span.ParentSpanID().HexString())
					buf.fieldString("spanID", span.SpanID().HexString())
					buf.fieldString("name", span.Name())
					buf.fieldString("kind", span.Kind().String())
					buf.fieldString("startTime", span.StartTimestamp().String())
					buf.fieldString("endTime", span.EndTimestamp().String())

					buf.fieldString("statusCode", span.Status().Code().String())
					buf.fieldString("statusMessage", span.Status().Message())

					buf.fieldAttrs("attributes", span.Attributes())

					buf.fieldSpanEvents("events", span.Events())
					buf.fieldSpanLinks("links", span.Links())
				})
			}
		}
	}

	return buf.buf.Bytes(), nil
}

func (b *dataBuffer) fieldSpanEvents(name string, events pdata.SpanEventSlice) {
	b.field(name, func() {
		b.spanEvents(events)
	})
}

func (b *dataBuffer) spanEvents(events pdata.SpanEventSlice) {
	b.array(func() {
		for i := 0; i < events.Len(); i++ {
			event := events.At(i)
			b.object(func() {
				b.fieldString("name", event.Name())
				b.fieldTime("timestamp", event.Timestamp())
				b.fieldUint32("droppedAttributesCount", event.DroppedAttributesCount())
				b.fieldAttrs("attributes", event.Attributes())
			})
		}
	})
}

func (b *dataBuffer) fieldSpanLinks(name string, links pdata.SpanLinkSlice) {
	b.field(name, func() {
		b.spanLinks(links)
	})
}

func (b *dataBuffer) spanLinks(links pdata.SpanLinkSlice) {
	b.array(func() {
		for i := 0; i < links.Len(); i++ {
			link := links.At(i)
			b.object(func() {
				b.fieldString("traceID", link.TraceID().HexString())
				b.fieldString("linkID", link.SpanID().HexString())
				b.fieldString("traceState", string(link.TraceState()))
				b.fieldUint32("droppedAttributesCount", link.DroppedAttributesCount())
				b.fieldAttrs("attributes", link.Attributes())
			})
		}
	})
}
