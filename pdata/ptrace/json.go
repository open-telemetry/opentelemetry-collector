// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptrace // import "go.opentelemetry.io/collector/pdata/ptrace"

import (
	"fmt"
	"slices"

	jsoniter "github.com/json-iterator/go"

	"go.opentelemetry.io/collector/pdata/internal"
	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
	"go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
)

// JSONMarshaler marshals pdata.Traces to JSON bytes using the OTLP/JSON format.
type JSONMarshaler struct{}

// MarshalTraces to the OTLP/JSON format.
func (*JSONMarshaler) MarshalTraces(td Traces) ([]byte, error) {
	dest := json.BorrowStream(nil)
	defer json.ReturnStream(dest)
	td.marshalJSONStream(dest)
	return slices.Clone(dest.Buffer()), dest.Error
}

// JSONUnmarshaler unmarshals OTLP/JSON formatted-bytes to pdata.Traces.
type JSONUnmarshaler struct{}

// UnmarshalTraces from OTLP/JSON format into pdata.Traces.
func (*JSONUnmarshaler) UnmarshalTraces(buf []byte) (Traces, error) {
	iter := jsoniter.ConfigFastest.BorrowIterator(buf)
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	td := NewTraces()
	td.unmarshalJSONIter(iter)
	if iter.Error != nil {
		return Traces{}, iter.Error
	}
	otlp.MigrateTraces(td.getOrig().ResourceSpans)
	return td, nil
}

func (ms ResourceSpans) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "resource":
			internal.UnmarshalJSONIterResource(internal.NewResource(&ms.orig.Resource, ms.state), iter)
		case "scopeSpans", "scope_spans":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.ScopeSpans().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "schemaUrl", "schema_url":
			ms.orig.SchemaUrl = iter.ReadString()
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms ScopeSpans) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "scope":
			internal.UnmarshalJSONIterInstrumentationScope(internal.NewInstrumentationScope(&ms.orig.Scope, ms.state), iter)
		case "spans":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.Spans().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "schemaUrl", "schema_url":
			ms.orig.SchemaUrl = iter.ReadString()
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms Span) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "traceId", "trace_id":
			if err := ms.orig.TraceId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
				iter.ReportError("readSpan.traceId", fmt.Sprintf("parse trace_id:%v", err))
			}
		case "spanId", "span_id":
			if err := ms.orig.SpanId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
				iter.ReportError("readSpan.spanId", fmt.Sprintf("parse span_id:%v", err))
			}
		case "traceState", "trace_state":
			ms.TraceState().FromRaw(iter.ReadString())
		case "parentSpanId", "parent_span_id":
			if err := ms.orig.ParentSpanId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
				iter.ReportError("readSpan.parentSpanId", fmt.Sprintf("parse parent_span_id:%v", err))
			}
		case "flags":
			ms.orig.Flags = json.ReadUint32(iter)
		case "name":
			ms.orig.Name = iter.ReadString()
		case "kind":
			ms.orig.Kind = otlptrace.Span_SpanKind(json.ReadEnumValue(iter, otlptrace.Span_SpanKind_value))
		case "startTimeUnixNano", "start_time_unix_nano":
			ms.orig.StartTimeUnixNano = json.ReadUint64(iter)
		case "endTimeUnixNano", "end_time_unix_nano":
			ms.orig.EndTimeUnixNano = json.ReadUint64(iter)
		case "attributes":
			internal.UnmarshalJSONIterMap(internal.NewMap(&ms.orig.Attributes, ms.state), iter)
		case "droppedAttributesCount", "dropped_attributes_count":
			ms.orig.DroppedAttributesCount = json.ReadUint32(iter)
		case "events":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.Events().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "droppedEventsCount", "dropped_events_count":
			ms.orig.DroppedEventsCount = json.ReadUint32(iter)
		case "links":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.Links().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "droppedLinksCount", "dropped_links_count":
			ms.orig.DroppedLinksCount = json.ReadUint32(iter)
		case "status":
			ms.Status().unmarshalJSONIter(iter)
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms Status) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "message":
			ms.orig.Message = iter.ReadString()
		case "code":
			ms.orig.Code = otlptrace.Status_StatusCode(json.ReadEnumValue(iter, otlptrace.Status_StatusCode_value))
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms SpanLink) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "traceId", "trace_id":
			if err := ms.orig.TraceId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
				iter.ReportError("readSpanLink", fmt.Sprintf("parse trace_id:%v", err))
			}
		case "spanId", "span_id":
			if err := ms.orig.SpanId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
				iter.ReportError("readSpanLink", fmt.Sprintf("parse span_id:%v", err))
			}
		case "traceState", "trace_state":
			ms.orig.TraceState = iter.ReadString()
		case "attributes":
			internal.UnmarshalJSONIterMap(internal.NewMap(&ms.orig.Attributes, ms.state), iter)
		case "droppedAttributesCount", "dropped_attributes_count":
			ms.orig.DroppedAttributesCount = json.ReadUint32(iter)
		case "flags":
			ms.orig.Flags = json.ReadUint32(iter)
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms SpanEvent) unmarshalJSONIter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "timeUnixNano", "time_unix_nano":
			ms.orig.TimeUnixNano = json.ReadUint64(iter)
		case "name":
			ms.orig.Name = iter.ReadString()
		case "attributes":
			internal.UnmarshalJSONIterMap(internal.NewMap(&ms.orig.Attributes, ms.state), iter)
		case "droppedAttributesCount", "dropped_attributes_count":
			ms.orig.DroppedAttributesCount = json.ReadUint32(iter)
		default:
			iter.Skip()
		}
		return true
	})
}
