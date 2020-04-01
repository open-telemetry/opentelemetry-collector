// Copyright 2019 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internaldata

import (
	"fmt"
	"strings"

	octrace "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
	tracetranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
)

const sourceFormat = "otlp_trace"

var (
	defaultProcessID = 0
)

func TraceDataToOC(td data.TraceData) []consumerdata.TraceData {
	resourceSpans := td.ResourceSpans()

	if resourceSpans.Len() == 0 {
		return nil
	}

	ocResourceSpansList := make([]consumerdata.TraceData, 0, resourceSpans.Len())

	for i := 0; i < resourceSpans.Len(); i++ {
		ocResourceSpansList = append(ocResourceSpansList, ResourceSpansToOC(resourceSpans.Get(i)))
	}

	return ocResourceSpansList
}

func ResourceSpansToOC(rs data.ResourceSpans) consumerdata.TraceData {
	ocTraceData := consumerdata.TraceData{
		SourceFormat: sourceFormat,
	}
	ocTraceData.Node, ocTraceData.Resource = internalResourceToOC(rs.Resource())
	ilss := rs.InstrumentationLibrarySpans()
	if ilss.Len() == 0 {
		return ocTraceData
	}
	// Approximate the number of the metrics as the number of the metrics in the first
	// instrumentation library info.
	ocSpans := make([]*octrace.Span, 0, ilss.Get(0).Spans().Len())
	for i := 0; i < ilss.Len(); i++ {
		// TODO: Handle instrumentation library name and version.
		spans := ilss.Get(i).Spans()
		for j := 0; j < spans.Len(); j++ {
			ocSpans = append(ocSpans, spanToOC(spans.Get(j)))
		}
	}
	ocTraceData.Spans = ocSpans
	return ocTraceData
}

func spanToOC(span data.Span) *octrace.Span {
	attributes := attributesMapToOCSpanAttributes(span.Attributes(), span.DroppedAttributesCount())
	if kindAttr := spanKindToOCAttribute(span.Kind()); kindAttr != nil {
		if attributes == nil {
			attributes = &octrace.Span_Attributes{
				AttributeMap:           make(map[string]*octrace.AttributeValue, 1),
				DroppedAttributesCount: 0,
			}
		}
		attributes.AttributeMap[tracetranslator.TagSpanKind] = kindAttr
	}

	return &octrace.Span{
		TraceId:      span.TraceID().Bytes(),
		SpanId:       span.SpanID().Bytes(),
		Tracestate:   traceStateToOC(span.TraceState()),
		ParentSpanId: span.ParentSpanID().Bytes(),
		Name: &octrace.TruncatableString{
			Value: span.Name(),
		},
		Kind:           spanKindToOC(span.Kind()),
		StartTime:      internal.UnixNanoToTimestamp(span.StartTime()),
		EndTime:        internal.UnixNanoToTimestamp(span.EndTime()),
		Attributes:     attributes,
		TimeEvents:     eventsToOC(span.Events(), span.DroppedEventsCount()),
		Links:          linksToOC(span.Links(), span.DroppedLinksCount()),
		Status:         statusToOC(span.Status()),
		ChildSpanCount: nil, // TODO(dmitryax): Handle once OTLP supports it
	}
}

func attributesMapToOCSpanAttributes(attributes data.AttributeMap, droppedCount uint32) *octrace.Span_Attributes {
	if attributes.Len() == 0 && droppedCount == 0 {
		return nil
	}

	return &octrace.Span_Attributes{
		AttributeMap:           attributesMapToOCAttributeMap(attributes),
		DroppedAttributesCount: int32(droppedCount),
	}
}

func attributesMapToOCAttributeMap(attributes data.AttributeMap) map[string]*octrace.AttributeValue {
	if attributes.Len() == 0 {
		return nil
	}

	ocAttributes := make(map[string]*octrace.AttributeValue, attributes.Len())
	for i := 0; i < attributes.Len(); i++ {
		attr := attributes.GetAttribute(i)
		ocAttributes[attr.Key()] = attributeValueToOC(attr)
	}
	return ocAttributes
}

func attributeValueToOC(attr data.AttributeKeyValue) *octrace.AttributeValue {
	a := &octrace.AttributeValue{}

	switch attr.ValType() {
	case data.AttributeValueSTRING:
		a.Value = &octrace.AttributeValue_StringValue{
			StringValue: &octrace.TruncatableString{
				Value: attr.StringVal(),
			},
		}
	case data.AttributeValueBOOL:
		a.Value = &octrace.AttributeValue_BoolValue{
			BoolValue: attr.BoolVal(),
		}
	case data.AttributeValueDOUBLE:
		a.Value = &octrace.AttributeValue_DoubleValue{
			DoubleValue: attr.DoubleVal(),
		}
	case data.AttributeValueINT:
		a.Value = &octrace.AttributeValue_IntValue{
			IntValue: attr.IntVal(),
		}
	default:
		a.Value = &octrace.AttributeValue_StringValue{
			StringValue: &octrace.TruncatableString{
				Value: fmt.Sprintf("<Unknown OpenTelemetry attribute value type %q>", attr.ValType()),
			},
		}
	}

	return a
}

func spanKindToOCAttribute(kind data.SpanKind) *octrace.AttributeValue {
	var ocKind tracetranslator.OpenTracingSpanKind
	switch kind {
	case data.SpanKindCONSUMER:
		ocKind = tracetranslator.OpenTracingSpanKindConsumer
	case data.SpanKindPRODUCER:
		ocKind = tracetranslator.OpenTracingSpanKindProducer
	case data.SpanKindUNSPECIFIED:
	case data.SpanKindINTERNAL:
	case data.SpanKindSERVER: // explicitly handled as SpanKind
	case data.SpanKindCLIENT: // explicitly handled as SpanKind
	default:

	}

	if string(ocKind) == "" {
		// No matching kind attribute value
		return nil
	}

	return stringAttributeValue(string(ocKind))
}

func stringAttributeValue(val string) *octrace.AttributeValue {
	return &octrace.AttributeValue{
		Value: &octrace.AttributeValue_StringValue{
			StringValue: &octrace.TruncatableString{
				Value: val,
			},
		},
	}
}

// OTLP follows the W3C format, e.g. "vendorname1=opaqueValue1,vendorname2=opaqueValue2"
func traceStateToOC(traceState data.TraceState) *octrace.Span_Tracestate {
	if traceState == "" {
		return nil
	}

	// key-value pairs in the "key1=value1" format
	pairs := strings.Split(string(traceState), ",")

	entries := make([]*octrace.Span_Tracestate_Entry, 0, len(pairs))
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 0 {
			continue
		}

		key := kv[0]
		val := ""
		if len(kv) >= 2 {
			val = kv[1]
		}

		entries = append(entries, &octrace.Span_Tracestate_Entry{
			Key:   key,
			Value: val,
		})
	}

	return &octrace.Span_Tracestate{
		Entries: entries,
	}
}

func spanKindToOC(kind data.SpanKind) octrace.Span_SpanKind {
	switch kind {
	case data.SpanKindSERVER:
		return octrace.Span_SERVER
	case data.SpanKindCLIENT:
		return octrace.Span_CLIENT
	// NOTE: see `spanKindToOCAttribute` function for custom kinds
	case data.SpanKindUNSPECIFIED:
	case data.SpanKindINTERNAL:
	case data.SpanKindPRODUCER:
	case data.SpanKindCONSUMER:
	default:
	}

	return octrace.Span_SPAN_KIND_UNSPECIFIED
}

func eventsToOC(events data.SpanEventSlice, droppedCount uint32) *octrace.Span_TimeEvents {
	if events.Len() == 0 {
		if droppedCount == 0 {
			return nil
		}
		return &octrace.Span_TimeEvents{
			TimeEvent:                 nil,
			DroppedMessageEventsCount: int32(droppedCount),
		}
	}

	ocEvents := make([]*octrace.Span_TimeEvent, 0, events.Len())
	for i := 0; i < events.Len(); i++ {
		ocEvents = append(ocEvents, eventToOC(events.Get(i)))
	}

	return &octrace.Span_TimeEvents{
		TimeEvent:               ocEvents,
		DroppedAnnotationsCount: int32(droppedCount),
	}
}

func eventToOC(event data.SpanEvent) *octrace.Span_TimeEvent {
	attrs := event.Attributes()

	// Consider TimeEvent to be of MessageEvent type if all and only relevant attributes are set
	ocMessageEventAttrs := []string{
		conventions.OCTimeEventMessageEventType,
		conventions.OCTimeEventMessageEventID,
		conventions.OCTimeEventMessageEventUSize,
		conventions.OCTimeEventMessageEventCSize,
	}
	if attrs.Len() == len(ocMessageEventAttrs) {
		ocMessageEventAttrValues := map[string]data.AttributeKeyValue{}
		var ocMessageEventAttrFound bool
		for _, attr := range ocMessageEventAttrs {
			akv, found := attrs.Get(attr)
			if found {
				ocMessageEventAttrFound = true
			}
			ocMessageEventAttrValues[attr] = akv
		}
		if ocMessageEventAttrFound {
			ocMessageEventType := ocMessageEventAttrValues[conventions.OCTimeEventMessageEventType]
			ocMessageEventTypeVal := octrace.Span_TimeEvent_MessageEvent_Type_value[ocMessageEventType.StringVal()]
			return &octrace.Span_TimeEvent{
				Time: internal.UnixNanoToTimestamp(event.Timestamp()),
				Value: &octrace.Span_TimeEvent_MessageEvent_{
					MessageEvent: &octrace.Span_TimeEvent_MessageEvent{
						Type:             octrace.Span_TimeEvent_MessageEvent_Type(ocMessageEventTypeVal),
						Id:               uint64(ocMessageEventAttrValues[conventions.OCTimeEventMessageEventID].IntVal()),
						UncompressedSize: uint64(ocMessageEventAttrValues[conventions.OCTimeEventMessageEventUSize].IntVal()),
						CompressedSize:   uint64(ocMessageEventAttrValues[conventions.OCTimeEventMessageEventCSize].IntVal()),
					},
				},
			}
		}
	}

	ocAttributes := attributesMapToOCSpanAttributes(attrs, event.DroppedAttributesCount())
	return &octrace.Span_TimeEvent{
		Time: internal.UnixNanoToTimestamp(event.Timestamp()),
		Value: &octrace.Span_TimeEvent_Annotation_{
			Annotation: &octrace.Span_TimeEvent_Annotation{
				Description: &octrace.TruncatableString{
					Value: event.Name(),
				},
				Attributes: ocAttributes,
			},
		},
	}
}

func linksToOC(links data.SpanLinkSlice, droppedCount uint32) *octrace.Span_Links {
	if links.Len() == 0 {
		if droppedCount == 0 {
			return nil
		}
		return &octrace.Span_Links{
			Link:              nil,
			DroppedLinksCount: int32(droppedCount),
		}
	}

	ocLinks := make([]*octrace.Span_Link, 0, links.Len())
	for i := 0; i < links.Len(); i++ {
		link := links.Get(i)
		ocLink := &octrace.Span_Link{
			TraceId:    link.TraceID().Bytes(),
			SpanId:     link.SpanID().Bytes(),
			Tracestate: traceStateToOC(link.TraceState()),
			Attributes: attributesMapToOCSpanAttributes(link.Attributes(), link.DroppedAttributesCount()),
		}
		ocLinks = append(ocLinks, ocLink)
	}

	return &octrace.Span_Links{
		Link:              ocLinks,
		DroppedLinksCount: int32(droppedCount),
	}
}

func statusToOC(status data.SpanStatus) *octrace.Status {
	if status.IsNil() {
		return nil
	}
	return &octrace.Status{
		Code:    int32(status.Code()),
		Message: status.Message(),
	}
}
