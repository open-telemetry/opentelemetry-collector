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

package internaldata

import (
	"fmt"
	"strings"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	octrace "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/occonventions"
	"go.opentelemetry.io/collector/translator/conventions"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

var (
	defaultProcessID = 0
)

// ResourceSpansToOC may be used only by OpenCensus receiver and exporter implementations.
// Deprecated: Use pdata.Traces.
// TODO: move this function to OpenCensus package.
func ResourceSpansToOC(rs pdata.ResourceSpans) (*occommon.Node, *ocresource.Resource, []*octrace.Span) {
	node, resource := internalResourceToOC(rs.Resource())
	ilss := rs.InstrumentationLibrarySpans()
	if ilss.Len() == 0 {
		return node, resource, nil
	}
	// Approximate the number of the spans as the number of the spans in the first
	// instrumentation library info.
	ocSpans := make([]*octrace.Span, 0, ilss.At(0).Spans().Len())
	for i := 0; i < ilss.Len(); i++ {
		ils := ilss.At(i)
		// TODO: Handle instrumentation library name and version.
		spans := ils.Spans()
		for j := 0; j < spans.Len(); j++ {
			ocSpans = append(ocSpans, spanToOC(spans.At(j)))
		}
	}
	return node, resource, ocSpans
}

func spanToOC(span pdata.Span) *octrace.Span {
	spaps := attributesMapToOCSameProcessAsParentSpan(span.Attributes())
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

	ocStatus, statusAttr := statusToOC(span.Status())
	if statusAttr != nil {
		if attributes == nil {
			attributes = &octrace.Span_Attributes{
				AttributeMap:           make(map[string]*octrace.AttributeValue, 1),
				DroppedAttributesCount: 0,
			}
		}
		attributes.AttributeMap[tracetranslator.TagStatusCode] = statusAttr
	}

	return &octrace.Span{
		TraceId:                 traceIDToOC(span.TraceID()),
		SpanId:                  spanIDToOC(span.SpanID()),
		Tracestate:              traceStateToOC(span.TraceState()),
		ParentSpanId:            spanIDToOC(span.ParentSpanID()),
		Name:                    stringToTruncatableString(span.Name()),
		Kind:                    spanKindToOC(span.Kind()),
		StartTime:               timestampAsTimestampPb(span.StartTimestamp()),
		EndTime:                 timestampAsTimestampPb(span.EndTimestamp()),
		Attributes:              attributes,
		TimeEvents:              eventsToOC(span.Events(), span.DroppedEventsCount()),
		Links:                   linksToOC(span.Links(), span.DroppedLinksCount()),
		Status:                  ocStatus,
		ChildSpanCount:          nil, // TODO(dmitryax): Handle once OTLP supports it
		SameProcessAsParentSpan: spaps,
	}
}

func attributesMapToOCSpanAttributes(attributes pdata.AttributeMap, droppedCount uint32) *octrace.Span_Attributes {
	if attributes.Len() == 0 && droppedCount == 0 {
		return nil
	}

	return &octrace.Span_Attributes{
		AttributeMap:           attributesMapToOCAttributeMap(attributes),
		DroppedAttributesCount: int32(droppedCount),
	}
}

func attributesMapToOCAttributeMap(attributes pdata.AttributeMap) map[string]*octrace.AttributeValue {
	if attributes.Len() == 0 {
		return nil
	}

	ocAttributes := make(map[string]*octrace.AttributeValue, attributes.Len())
	attributes.Range(func(k string, v pdata.AttributeValue) bool {
		ocAttributes[k] = attributeValueToOC(v)
		return true
	})
	return ocAttributes
}

func attributeValueToOC(attr pdata.AttributeValue) *octrace.AttributeValue {
	a := &octrace.AttributeValue{}

	switch attr.Type() {
	case pdata.AttributeValueTypeString:
		a.Value = &octrace.AttributeValue_StringValue{
			StringValue: stringToTruncatableString(attr.StringVal()),
		}
	case pdata.AttributeValueTypeBool:
		a.Value = &octrace.AttributeValue_BoolValue{
			BoolValue: attr.BoolVal(),
		}
	case pdata.AttributeValueTypeDouble:
		a.Value = &octrace.AttributeValue_DoubleValue{
			DoubleValue: attr.DoubleVal(),
		}
	case pdata.AttributeValueTypeInt:
		a.Value = &octrace.AttributeValue_IntValue{
			IntValue: attr.IntVal(),
		}
	case pdata.AttributeValueTypeMap:
		a.Value = &octrace.AttributeValue_StringValue{
			StringValue: stringToTruncatableString(tracetranslator.AttributeValueToString(attr)),
		}
	case pdata.AttributeValueTypeArray:
		a.Value = &octrace.AttributeValue_StringValue{
			StringValue: stringToTruncatableString(tracetranslator.AttributeValueToString(attr)),
		}
	default:
		a.Value = &octrace.AttributeValue_StringValue{
			StringValue: stringToTruncatableString(fmt.Sprintf("<Unknown OpenTelemetry attribute value type %q>", attr.Type())),
		}
	}

	return a
}

func spanKindToOCAttribute(kind pdata.SpanKind) *octrace.AttributeValue {
	var ocKind tracetranslator.OpenTracingSpanKind
	switch kind {
	case pdata.SpanKindConsumer:
		ocKind = tracetranslator.OpenTracingSpanKindConsumer
	case pdata.SpanKindProducer:
		ocKind = tracetranslator.OpenTracingSpanKindProducer
	case pdata.SpanKindInternal:
		ocKind = tracetranslator.OpenTracingSpanKindInternal
	case pdata.SpanKindUnspecified:
	case pdata.SpanKindServer: // explicitly handled as SpanKind
	case pdata.SpanKindClient: // explicitly handled as SpanKind
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
			StringValue: stringToTruncatableString(val),
		},
	}
}

func attributesMapToOCSameProcessAsParentSpan(attr pdata.AttributeMap) *wrapperspb.BoolValue {
	val, ok := attr.Get(occonventions.AttributeSameProcessAsParentSpan)
	if !ok || val.Type() != pdata.AttributeValueTypeBool {
		return nil
	}
	return wrapperspb.Bool(val.BoolVal())
}

// OTLP follows the W3C format, e.g. "vendorname1=opaqueValue1,vendorname2=opaqueValue2"
func traceStateToOC(traceState pdata.TraceState) *octrace.Span_Tracestate {
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

func spanKindToOC(kind pdata.SpanKind) octrace.Span_SpanKind {
	switch kind {
	case pdata.SpanKindServer:
		return octrace.Span_SERVER
	case pdata.SpanKindClient:
		return octrace.Span_CLIENT
	// NOTE: see `spanKindToOCAttribute` function for custom kinds
	case pdata.SpanKindUnspecified:
	case pdata.SpanKindInternal:
	case pdata.SpanKindProducer:
	case pdata.SpanKindConsumer:
	default:
	}

	return octrace.Span_SPAN_KIND_UNSPECIFIED
}

func eventsToOC(events pdata.SpanEventSlice, droppedCount uint32) *octrace.Span_TimeEvents {
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
		ocEvents = append(ocEvents, eventToOC(events.At(i)))
	}

	return &octrace.Span_TimeEvents{
		TimeEvent:               ocEvents,
		DroppedAnnotationsCount: int32(droppedCount),
	}
}

func eventToOC(event pdata.SpanEvent) *octrace.Span_TimeEvent {
	attrs := event.Attributes()

	// Consider TimeEvent to be of MessageEvent type if all and only relevant attributes are set
	ocMessageEventAttrs := []string{
		conventions.AttributeMessageType,
		conventions.AttributeMessageID,
		conventions.AttributeMessageUncompressedSize,
		conventions.AttributeMessageCompressedSize,
	}
	// TODO: Find a better way to check for message_event. Maybe use the event.Name.
	if attrs.Len() == len(ocMessageEventAttrs) {
		ocMessageEventAttrValues := map[string]pdata.AttributeValue{}
		var ocMessageEventAttrFound bool
		for _, attr := range ocMessageEventAttrs {
			akv, found := attrs.Get(attr)
			if found {
				ocMessageEventAttrFound = true
			}
			ocMessageEventAttrValues[attr] = akv
		}
		if ocMessageEventAttrFound {
			ocMessageEventType := ocMessageEventAttrValues[conventions.AttributeMessageType]
			ocMessageEventTypeVal := octrace.Span_TimeEvent_MessageEvent_Type_value[ocMessageEventType.StringVal()]
			return &octrace.Span_TimeEvent{
				Time: timestampAsTimestampPb(event.Timestamp()),
				Value: &octrace.Span_TimeEvent_MessageEvent_{
					MessageEvent: &octrace.Span_TimeEvent_MessageEvent{
						Type:             octrace.Span_TimeEvent_MessageEvent_Type(ocMessageEventTypeVal),
						Id:               uint64(ocMessageEventAttrValues[conventions.AttributeMessageID].IntVal()),
						UncompressedSize: uint64(ocMessageEventAttrValues[conventions.AttributeMessageUncompressedSize].IntVal()),
						CompressedSize:   uint64(ocMessageEventAttrValues[conventions.AttributeMessageCompressedSize].IntVal()),
					},
				},
			}
		}
	}

	ocAttributes := attributesMapToOCSpanAttributes(attrs, event.DroppedAttributesCount())
	return &octrace.Span_TimeEvent{
		Time: timestampAsTimestampPb(event.Timestamp()),
		Value: &octrace.Span_TimeEvent_Annotation_{
			Annotation: &octrace.Span_TimeEvent_Annotation{
				Description: stringToTruncatableString(event.Name()),
				Attributes:  ocAttributes,
			},
		},
	}
}

func linksToOC(links pdata.SpanLinkSlice, droppedCount uint32) *octrace.Span_Links {
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
		link := links.At(i)
		ocLink := &octrace.Span_Link{
			TraceId:    traceIDToOC(link.TraceID()),
			SpanId:     spanIDToOC(link.SpanID()),
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

func traceIDToOC(tid pdata.TraceID) []byte {
	if tid.IsEmpty() {
		return nil
	}
	tidBytes := tid.Bytes()
	return tidBytes[:]
}

func spanIDToOC(sid pdata.SpanID) []byte {
	if sid.IsEmpty() {
		return nil
	}
	sidBytes := sid.Bytes()
	return sidBytes[:]
}

func statusToOC(status pdata.SpanStatus) (*octrace.Status, *octrace.AttributeValue) {
	var attr *octrace.AttributeValue
	var oc int32
	switch status.Code() {
	case pdata.StatusCodeUnset:
		// Unset in OTLP corresponds to OK in OpenCensus.
		oc = trace.StatusCodeOK
	case pdata.StatusCodeOk:
		// OK in OpenCensus is the closest to OK in OTLP.
		oc = trace.StatusCodeOK
		// We will also add an attribute to indicate that it is OTLP OK, different from OTLP Unset.
		attr = &octrace.AttributeValue{Value: &octrace.AttributeValue_IntValue{IntValue: int64(status.Code())}}
	case pdata.StatusCodeError:
		oc = trace.StatusCodeUnknown
	}

	return &octrace.Status{Code: oc, Message: status.Message()}, attr
}

func stringToTruncatableString(str string) *octrace.TruncatableString {
	if str == "" {
		return nil
	}
	return &octrace.TruncatableString{
		Value: str,
	}
}
