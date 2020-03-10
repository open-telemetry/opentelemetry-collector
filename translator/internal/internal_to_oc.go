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

package internal

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	octrace "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
	tracetranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
)

const sourceFormat = "ot_internal_trace"

var (
	defaultTime      = time.Time{} // zero
	defaultTimestamp = timestamp.Timestamp{}
	defaultProcessID = 0
)

func internalToOC(td data.TraceData) consumerdata.TraceData {
	ocTraceData := consumerdata.TraceData{
		SourceFormat: sourceFormat,
	}

	resourceSpansList := td.ResourceSpans()

	if len(resourceSpansList) == 0 {
		return ocTraceData
	}

	// Original resource with its spans stored in the first object of resourceSpansList
	// Additional spans originated from different resources stored in the following objects of resourceSpansList
	originalResourceSpans := resourceSpansList[0]
	additionalResourceSpansList := resourceSpansList[1:]

	ocTraceData.Node, ocTraceData.Resource = internalResourceToOC(originalResourceSpans.Resource())
	ocSpans := make([]*octrace.Span, 0, td.SpanCount())

	for _, span := range originalResourceSpans.Spans() {
		ocSpans = append(ocSpans, spanToOC(span))
	}
	for _, resourceSpans := range additionalResourceSpansList {
		_, resource := internalResourceToOC(resourceSpans.Resource())
		for _, span := range resourceSpans.Spans() {
			ocSpan := spanToOC(span)
			ocSpan.Resource = resource
			ocSpans = append(ocSpans, ocSpan)
		}
	}
	ocTraceData.Spans = ocSpans

	return ocTraceData
}

func internalResourceToOC(resource *data.Resource) (*occommon.Node, *ocresource.Resource) {
	if resource == nil {
		return nil, nil
	}

	attrs := resource.Attributes()

	ocNode := occommon.Node{}
	ocResource := ocresource.Resource{}

	if len(attrs) == 0 {
		return &ocNode, &ocResource
	}

	labels := make(map[string]string, len(attrs))
	for key, attributeValue := range attrs {
		val := attributeValueToString(attributeValue)

		switch key {
		case conventions.OCAttributeResourceType:
			ocResource.Type = val
		case conventions.AttributeServiceName:
			if ocNode.ServiceInfo == nil {
				ocNode.ServiceInfo = &occommon.ServiceInfo{}
			}
			ocNode.ServiceInfo.Name = val
		case conventions.OCAttributeProcessStartTime:
			t, err := time.Parse(time.RFC3339Nano, val)
			if err != nil {
				t = defaultTime
			}
			ts, err := ptypes.TimestampProto(t)
			if err != nil {
				ts = &defaultTimestamp
			}
			if ocNode.Identifier == nil {
				ocNode.Identifier = &occommon.ProcessIdentifier{}
			}
			ocNode.Identifier.StartTimestamp = ts
		case conventions.AttributeHostHostname:
			if ocNode.Identifier == nil {
				ocNode.Identifier = &occommon.ProcessIdentifier{}
			}
			ocNode.Identifier.HostName = val
		case conventions.OCAttributeProcessID:
			pid, err := strconv.Atoi(val)
			if err != nil {
				pid = defaultProcessID
			}
			if ocNode.Identifier == nil {
				ocNode.Identifier = &occommon.ProcessIdentifier{}
			}
			ocNode.Identifier.Pid = uint32(pid)
		case conventions.AttributeLibraryVersion:
			if ocNode.LibraryInfo == nil {
				ocNode.LibraryInfo = &occommon.LibraryInfo{}
			}
			ocNode.LibraryInfo.CoreLibraryVersion = val
		case conventions.OCAttributeExporterVersion:
			if ocNode.LibraryInfo == nil {
				ocNode.LibraryInfo = &occommon.LibraryInfo{}
			}
			ocNode.LibraryInfo.ExporterVersion = val
		case conventions.AttributeLibraryLanguage:
			if code, ok := occommon.LibraryInfo_Language_value[val]; ok {
				if ocNode.LibraryInfo == nil {
					ocNode.LibraryInfo = &occommon.LibraryInfo{}
				}
				ocNode.LibraryInfo.Language = occommon.LibraryInfo_Language(code)
			}
		default:
			// Not a special attribute, put it into resource labels
			labels[key] = val
		}
	}
	ocResource.Labels = labels

	return &ocNode, &ocResource
}

func attributeValueToString(attr data.AttributeValue) string {
	switch attr.Type() {
	case data.AttributeValueSTRING:
		return attr.StringVal()
	case data.AttributeValueBOOL:
		return strconv.FormatBool(attr.BoolVal())
	case data.AttributeValueDOUBLE:
		return strconv.FormatFloat(attr.DoubleVal(), 'f', -1, 64)
	case data.AttributeValueINT:
		return strconv.FormatInt(attr.IntVal(), 10)
	default:
		return fmt.Sprintf("<Unknown OpenTelemetry attribute value type %q>", attr.Type())
	}
}

func spanToOC(span *data.Span) *octrace.Span {
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
		TraceId:      span.TraceIDBytes(),
		SpanId:       span.SpanIDBytes(),
		Tracestate:   traceStateToOC(span.TraceState()),
		ParentSpanId: span.ParentSpanIDBytes(),
		Name: &octrace.TruncatableString{
			Value: span.Name(),
		},
		Kind:           spanKindToOC(span.Kind()),
		StartTime:      internal.UnixnanoToTimestamp(uint64(span.StartTime())),
		EndTime:        internal.UnixnanoToTimestamp(uint64(span.EndTime())),
		Attributes:     attributes,
		TimeEvents:     eventsToOC(span.Events(), span.DroppedEventsCount()),
		Links:          linksToOC(span.Links(), span.DroppedLinksCount()),
		Status:         statusToOC(span.Status()),
		ChildSpanCount: nil, // TODO(dmitryax): Handle once OTLP supports it
	}
}

func attributesMapToOCSpanAttributes(attributes data.AttributesMap, droppedCount uint32) *octrace.Span_Attributes {
	if len(attributes) == 0 && droppedCount == 0 {
		return nil
	}

	return &octrace.Span_Attributes{
		AttributeMap:           attributesMapToOCAttributeMap(attributes),
		DroppedAttributesCount: int32(droppedCount),
	}
}

func attributesMapToOCAttributeMap(attributes data.AttributesMap) map[string]*octrace.AttributeValue {
	if len(attributes) == 0 {
		return nil
	}

	ocAttributes := make(map[string]*octrace.AttributeValue, len(attributes))
	for key, attr := range attributes {
		ocAttributes[key] = attributeValueToOC(attr)
	}
	return ocAttributes
}

func attributeValueToOC(attr data.AttributeValue) *octrace.AttributeValue {
	a := &octrace.AttributeValue{}

	switch attr.Type() {
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
				Value: fmt.Sprintf("<Unknown OpenTelemetry attribute value type %q>", attr.Type()),
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

func eventsToOC(events []*data.SpanEvent, droppedCount uint32) *octrace.Span_TimeEvents {
	if len(events) == 0 && droppedCount == 0 {
		return nil
	}

	ocEvents := make([]*octrace.Span_TimeEvent, 0, len(events))
	for _, event := range events {
		ocEvents = append(ocEvents, eventToOC(event))
	}

	return &octrace.Span_TimeEvents{
		TimeEvent:                 ocEvents,
		DroppedMessageEventsCount: int32(droppedCount),
	}
}

func eventToOC(event *data.SpanEvent) *octrace.Span_TimeEvent {
	attrs := event.Attributes()

	ocMessageEventType, ocMessageEventTypeAttrFound := attrs[conventions.OCTimeEventMessageEventType]
	ocMessageEventID, ocMessageEventIDAttrFound := attrs[conventions.OCTimeEventMessageEventID]
	ocMessageEventUSize, ocMessageEventUSizeAttrFound := attrs[conventions.OCTimeEventMessageEventUSize]
	ocMessageEventCSize, ocMessageEventCSizeAttrFound := attrs[conventions.OCTimeEventMessageEventCSize]

	// Consider TimeEvent to be of MessageEvent type if all relevant attributes are set
	if ocMessageEventTypeAttrFound && ocMessageEventIDAttrFound && ocMessageEventUSizeAttrFound && ocMessageEventCSizeAttrFound {
		ocMessageEventTypeVal, _ := octrace.Span_TimeEvent_MessageEvent_Type_value[ocMessageEventType.StringVal()]
		return &octrace.Span_TimeEvent{
			Time: internal.UnixnanoToTimestamp(uint64(event.Timestamp())),
			Value: &octrace.Span_TimeEvent_MessageEvent_{
				MessageEvent: &octrace.Span_TimeEvent_MessageEvent{
					Type:             octrace.Span_TimeEvent_MessageEvent_Type(ocMessageEventTypeVal),
					Id:               uint64(ocMessageEventID.IntVal()),
					UncompressedSize: uint64(ocMessageEventUSize.IntVal()),
					CompressedSize:   uint64(ocMessageEventCSize.IntVal()),
				},
			},
		}
	}

	ocAttributes := attributesMapToOCSpanAttributes(attrs, event.DroppedAttributesCount())
	return &octrace.Span_TimeEvent{
		Time: internal.UnixnanoToTimestamp(uint64(event.Timestamp())),
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

func linksToOC(links []*data.SpanLink, droppedCount uint32) *octrace.Span_Links {
	if len(links) == 0 && droppedCount == 0 {
		return nil
	}

	return &octrace.Span_Links{
		Link:              linksToOCLinks(links),
		DroppedLinksCount: int32(droppedCount),
	}
}

func linksToOCLinks(links []*data.SpanLink) []*octrace.Span_Link {
	if len(links) == 0 {
		return nil
	}

	ocLinks := make([]*octrace.Span_Link, 0, len(links))
	for _, link := range links {
		ocLink := &octrace.Span_Link{
			TraceId:    link.TraceIDBytes(),
			SpanId:     link.SpanIDBytes(),
			Tracestate: traceStateToOC(link.TraceState()),
			Attributes: attributesMapToOCSpanAttributes(link.Attributes(), link.DroppedAttributesCount()),
		}
		ocLinks = append(ocLinks, ocLink)
	}
	return ocLinks
}

func statusToOC(status data.SpanStatus) *octrace.Status {
	emptyStatus := data.SpanStatus{}
	if status == emptyStatus {
		return nil
	}
	return &octrace.Status{
		Code:    int32(status.Code()),
		Message: status.Message(),
	}
}
