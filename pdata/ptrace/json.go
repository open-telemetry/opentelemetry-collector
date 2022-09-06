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

package ptrace // import "go.opentelemetry.io/collector/pdata/ptrace"

import (
	"bytes"
	"fmt"

	"github.com/gogo/protobuf/jsonpb"
	jsoniter "github.com/json-iterator/go"

	"go.opentelemetry.io/collector/pdata/internal"
	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
	"go.opentelemetry.io/collector/pdata/internal/json"
)

// NewJSONMarshaler returns a model.Marshaler. Marshals to OTLP json bytes.
func NewJSONMarshaler() Marshaler {
	return newJSONMarshaler()
}

type jsonMarshaler struct {
	delegate jsonpb.Marshaler
}

func newJSONMarshaler() *jsonMarshaler {
	return &jsonMarshaler{delegate: jsonpb.Marshaler{}}
}

func (e *jsonMarshaler) MarshalTraces(td Traces) ([]byte, error) {
	buf := bytes.Buffer{}
	pb := internal.TracesToProto(internal.Traces(td))
	err := e.delegate.Marshal(&buf, &pb)
	return buf.Bytes(), err
}

// NewJSONUnmarshaler returns a model.Unmarshaler. Unmarshalls from OTLP json bytes.
func NewJSONUnmarshaler() Unmarshaler {
	return &jsonUnmarshaler{}
}

type jsonUnmarshaler struct {
}

func (d *jsonUnmarshaler) UnmarshalTraces(buf []byte) (Traces, error) {
	iter := jsoniter.ConfigFastest.BorrowIterator(buf)
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	td := readTraceData(iter)
	err := iter.Error
	return Traces(internal.TracesFromProto(td)), err
}

func readTraceData(iter *jsoniter.Iterator) otlptrace.TracesData {
	td := otlptrace.TracesData{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "resourceSpans", "resource_spans":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				td.ResourceSpans = append(td.ResourceSpans, readResourceSpans(iter))
				return true
			})
		default:
			iter.Skip()
		}
		return true
	})
	return td
}

func readResourceSpans(iter *jsoniter.Iterator) *otlptrace.ResourceSpans {
	rs := &otlptrace.ResourceSpans{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "resource":
			json.ReadResource(iter, &rs.Resource)
		case "scopeSpans", "scope_spans":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				rs.ScopeSpans = append(rs.ScopeSpans, readScopeSpans(iter))
				return true
			})
		case "schemaUrl", "schema_url":
			rs.SchemaUrl = iter.ReadString()
		default:
			iter.Skip()
		}
		return true
	})
	return rs
}

func readScopeSpans(iter *jsoniter.Iterator) *otlptrace.ScopeSpans {
	ils := &otlptrace.ScopeSpans{}

	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "scope":
			json.ReadScope(iter, &ils.Scope)
		case "spans":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ils.Spans = append(ils.Spans, readSpan(iter))
				return true
			})
		case "schemaUrl", "schema_url":
			ils.SchemaUrl = iter.ReadString()
		default:
			iter.Skip()
		}
		return true
	})
	return ils
}

func readSpan(iter *jsoniter.Iterator) *otlptrace.Span {
	sp := &otlptrace.Span{}

	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "traceId", "trace_id":
			if err := sp.TraceId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
				iter.ReportError("readSpan.traceId", fmt.Sprintf("parse trace_id:%v", err))
			}
		case "spanId", "span_id":
			if err := sp.SpanId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
				iter.ReportError("readSpan.spanId", fmt.Sprintf("parse span_id:%v", err))
			}
		case "traceState", "trace_state":
			sp.TraceState = iter.ReadString()
		case "parentSpanId", "parent_span_id":
			if err := sp.ParentSpanId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
				iter.ReportError("readSpan.parentSpanId", fmt.Sprintf("parse parent_span_id:%v", err))
			}
		case "name":
			sp.Name = iter.ReadString()
		case "kind":
			sp.Kind = otlptrace.Span_SpanKind(json.ReadEnumValue(iter, otlptrace.Span_SpanKind_value))
		case "startTimeUnixNano", "start_time_unix_nano":
			sp.StartTimeUnixNano = json.ReadUint64(iter)
		case "endTimeUnixNano", "end_time_unix_nano":
			sp.EndTimeUnixNano = json.ReadUint64(iter)
		case "attributes":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				sp.Attributes = append(sp.Attributes, json.ReadAttribute(iter))
				return true
			})
		case "droppedAttributesCount", "dropped_attributes_count":
			sp.DroppedAttributesCount = json.ReadUint32(iter)
		case "events":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				sp.Events = append(sp.Events, readSpanEvent(iter))
				return true
			})
		case "droppedEventsCount", "dropped_events_count":
			sp.DroppedEventsCount = json.ReadUint32(iter)
		case "links":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				sp.Links = append(sp.Links, readSpanLink(iter))
				return true
			})
		case "droppedLinksCount", "dropped_links_count":
			sp.DroppedLinksCount = json.ReadUint32(iter)
		case "status":
			iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
				switch f {
				case "message":
					sp.Status.Message = iter.ReadString()
				case "code":
					sp.Status.Code = otlptrace.Status_StatusCode(json.ReadEnumValue(iter, otlptrace.Status_StatusCode_value))
				default:
					iter.Skip()
				}
				return true
			})
		default:
			iter.Skip()
		}
		return true
	})
	return sp
}

func readSpanLink(iter *jsoniter.Iterator) *otlptrace.Span_Link {
	link := &otlptrace.Span_Link{}

	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "traceId", "trace_id":
			if err := link.TraceId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
				iter.ReportError("readSpanLink", fmt.Sprintf("parse trace_id:%v", err))
			}
		case "spanId", "span_id":
			if err := link.SpanId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
				iter.ReportError("readSpanLink", fmt.Sprintf("parse span_id:%v", err))
			}
		case "traceState", "trace_state":
			link.TraceState = iter.ReadString()
		case "attributes":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				link.Attributes = append(link.Attributes, json.ReadAttribute(iter))
				return true
			})
		case "droppedAttributesCount", "dropped_attributes_count":
			link.DroppedAttributesCount = json.ReadUint32(iter)
		default:
			iter.Skip()
		}
		return true
	})
	return link
}

func readSpanEvent(iter *jsoniter.Iterator) *otlptrace.Span_Event {
	event := &otlptrace.Span_Event{}

	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "timeUnixNano", "time_unix_nano":
			event.TimeUnixNano = json.ReadUint64(iter)
		case "name":
			event.Name = iter.ReadString()
		case "attributes":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				event.Attributes = append(event.Attributes, json.ReadAttribute(iter))
				return true
			})
		case "droppedAttributesCount", "dropped_attributes_count":
			event.DroppedAttributesCount = json.ReadUint32(iter)
		default:
			iter.Skip()
		}
		return true
	})
	return event
}
