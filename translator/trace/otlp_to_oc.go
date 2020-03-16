// Copyright 2020 OpenTelemetry Authors
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

package tracetranslator

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
	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	otlpresource "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"
	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

const sourceFormat = "otlp_trace"

var (
	defaultTime      = time.Time{} // zero
	defaultTimestamp = timestamp.Timestamp{}
	defaultProcessID = 0
)

// ResourceSpansToTraceData converts spans from OTLP to internal (OpenCensus) format
func ResourceSpansToTraceData(resourceSpans *otlptrace.ResourceSpans) consumerdata.TraceData {
	node, resource := resourceToOC(resourceSpans.Resource)
	td := consumerdata.TraceData{
		Node:         node,
		Resource:     resource,
		SourceFormat: sourceFormat,
	}

	if len(resourceSpans.InstrumentationLibrarySpans) == 0 {
		return td
	}

	// Allocate slice with a capacity approximated for the case when there is only one
	// InstrumentationLibrary or the first InstrumentationLibrary contains most of the data.
	// This is a best guess only that reduced slice re-allocations.
	spans := make([]*octrace.Span, 0, len(resourceSpans.InstrumentationLibrarySpans[0].Spans))
	for _, ils := range resourceSpans.InstrumentationLibrarySpans {
		for _, span := range ils.Spans {
			spans = append(spans, spanToOC(span))
		}
	}

	td.Spans = spans
	return td
}

func spanToOC(span *otlptrace.Span) *octrace.Span {
	attributes := attributesToOCSpanAttributes(span.Attributes, span.DroppedAttributesCount)
	if kindAttr := kindToAttribute(span.Kind); kindAttr != nil {
		if attributes == nil {
			attributes = &octrace.Span_Attributes{
				AttributeMap:           make(map[string]*octrace.AttributeValue, 1),
				DroppedAttributesCount: 0,
			}
		}
		attributes.AttributeMap[TagSpanKind] = kindAttr
	}

	return &octrace.Span{
		TraceId:      span.TraceId,
		SpanId:       span.SpanId,
		Tracestate:   tracestateToOC(span.TraceState),
		ParentSpanId: span.ParentSpanId,
		Name: &octrace.TruncatableString{
			Value: span.Name,
		},
		Kind:           kindToOC(span.Kind),
		StartTime:      internal.UnixnanoToTimestamp(data.TimestampUnixNano(span.StartTimeUnixnano)),
		EndTime:        internal.UnixnanoToTimestamp(data.TimestampUnixNano(span.EndTimeUnixnano)),
		Attributes:     attributes,
		TimeEvents:     eventsToOC(span.Events, span.DroppedEventsCount),
		Links:          linksToOC(span.Links, span.DroppedLinksCount),
		Status:         statusToOC(span.Status),
		ChildSpanCount: nil, // TODO(nilebox): Handle once OTLP supports it
	}
}

func linksToOC(links []*otlptrace.Span_Link, droppedCount uint32) *octrace.Span_Links {
	if len(links) == 0 && droppedCount == 0 {
		return nil
	}

	return &octrace.Span_Links{
		Link:              linksToOCLinks(links),
		DroppedLinksCount: int32(droppedCount),
	}
}

func linksToOCLinks(links []*otlptrace.Span_Link) []*octrace.Span_Link {
	if len(links) == 0 {
		return nil
	}

	ocLinks := make([]*octrace.Span_Link, 0, len(links))
	for _, link := range links {
		ocLink := &octrace.Span_Link{
			TraceId:    link.TraceId,
			SpanId:     link.SpanId,
			Tracestate: tracestateToOC(link.TraceState),
			Attributes: attributesToOCSpanAttributes(link.Attributes, link.DroppedAttributesCount),
		}
		ocLinks = append(ocLinks, ocLink)
	}
	return ocLinks
}

func eventsToOC(events []*otlptrace.Span_Event, droppedCount uint32) *octrace.Span_TimeEvents {
	if len(events) == 0 && droppedCount == 0 {
		return nil
	}

	return &octrace.Span_TimeEvents{
		TimeEvent:                 eventsToOCEvents(events),
		DroppedMessageEventsCount: int32(droppedCount),
	}
}

func eventsToOCEvents(events []*otlptrace.Span_Event) []*octrace.Span_TimeEvent {
	if len(events) == 0 {
		return nil
	}

	ocEvents := make([]*octrace.Span_TimeEvent, 0, len(events))
	for _, event := range events {
		ocAttributes := attributesToOCSpanAttributes(event.Attributes, event.DroppedAttributesCount)
		ocEvent := &octrace.Span_TimeEvent{
			Time: internal.UnixnanoToTimestamp(data.TimestampUnixNano(event.TimeUnixnano)),
			Value: &octrace.Span_TimeEvent_Annotation_{
				Annotation: &octrace.Span_TimeEvent_Annotation{
					Description: &octrace.TruncatableString{
						Value: event.Name,
					},
					Attributes: ocAttributes,
				},
			},
		}

		ocEvents = append(ocEvents, ocEvent)
	}
	return ocEvents
}

func statusToOC(status *otlptrace.Status) *octrace.Status {
	if status == nil {
		return nil
	}
	return &octrace.Status{
		Code:    int32(status.Code),
		Message: status.Message,
	}
}

func resourceToOC(resource *otlpresource.Resource) (*occommon.Node, *ocresource.Resource) {
	if resource == nil {
		return nil, nil
	}

	ocNode := occommon.Node{}
	ocResource := ocresource.Resource{}

	if len(resource.Attributes) == 0 {
		return &ocNode, &ocResource
	}

	labels := make(map[string]string, len(resource.Attributes))
	for _, attr := range resource.Attributes {
		key := attr.Key
		val := attributeValueToString(attr)

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

func attributesToOCSpanAttributes(attributes []*otlpcommon.AttributeKeyValue, droppedCount uint32) *octrace.Span_Attributes {
	if len(attributes) == 0 && droppedCount == 0 {
		return nil
	}

	return &octrace.Span_Attributes{
		AttributeMap:           attributesToOCMap(attributes),
		DroppedAttributesCount: int32(droppedCount),
	}
}

func attributesToOCMap(attributes []*otlpcommon.AttributeKeyValue) map[string]*octrace.AttributeValue {
	if len(attributes) == 0 {
		return nil
	}

	ocAttributes := make(map[string]*octrace.AttributeValue, len(attributes))
	for _, attr := range attributes {
		ocAttributes[attr.Key] = attributeValueToOC(attr)
	}
	return ocAttributes
}

func attributeValueToString(attr *otlpcommon.AttributeKeyValue) string {
	switch attr.Type {
	case otlpcommon.AttributeKeyValue_STRING:
		return attr.StringValue
	case otlpcommon.AttributeKeyValue_BOOL:
		return strconv.FormatBool(attr.BoolValue)
	case otlpcommon.AttributeKeyValue_DOUBLE:
		return strconv.FormatFloat(attr.DoubleValue, 'f', -1, 64)
	case otlpcommon.AttributeKeyValue_INT:
		return strconv.FormatInt(attr.IntValue, 10)
	default:
		return fmt.Sprintf("<Unknown OpenTelemetry attribute value type %q>", attr.Type)
	}
}

func attributeValueToOC(attr *otlpcommon.AttributeKeyValue) *octrace.AttributeValue {
	a := &octrace.AttributeValue{}

	switch attr.Type {
	case otlpcommon.AttributeKeyValue_STRING:
		a.Value = &octrace.AttributeValue_StringValue{
			StringValue: &octrace.TruncatableString{
				Value: attr.StringValue,
			},
		}
	case otlpcommon.AttributeKeyValue_BOOL:
		a.Value = &octrace.AttributeValue_BoolValue{
			BoolValue: attr.BoolValue,
		}
	case otlpcommon.AttributeKeyValue_DOUBLE:
		a.Value = &octrace.AttributeValue_DoubleValue{
			DoubleValue: attr.DoubleValue,
		}
	case otlpcommon.AttributeKeyValue_INT:
		a.Value = &octrace.AttributeValue_IntValue{
			IntValue: attr.IntValue,
		}
	default:
		a.Value = &octrace.AttributeValue_StringValue{
			StringValue: &octrace.TruncatableString{
				Value: fmt.Sprintf("<Unknown OpenTelemetry attribute value type %q>", attr.Type),
			},
		}
	}

	return a
}

// OTLP follows the W3C format, e.g. "vendorname1=opaqueValue1,vendorname2=opaqueValue2"
// See https://w3c.github.io/trace-context/#combined-header-value
func tracestateToOC(tracestate string) *octrace.Span_Tracestate {
	if tracestate == "" {
		return nil
	}

	// key-value pairs in the "key1=value1" format
	pairs := strings.Split(tracestate, ",")

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

func kindToOC(kind otlptrace.Span_SpanKind) octrace.Span_SpanKind {
	switch kind {
	case otlptrace.Span_SERVER:
		return octrace.Span_SERVER
	case otlptrace.Span_CLIENT:
		return octrace.Span_CLIENT
	// NOTE: see `kindToAttribute` function for custom kinds
	case otlptrace.Span_SPAN_KIND_UNSPECIFIED:
	case otlptrace.Span_INTERNAL:
	case otlptrace.Span_PRODUCER:
	case otlptrace.Span_CONSUMER:
	default:
	}

	return octrace.Span_SPAN_KIND_UNSPECIFIED
}

func kindToAttribute(kind otlptrace.Span_SpanKind) *octrace.AttributeValue {
	var ocKind OpenTracingSpanKind
	switch kind {
	case otlptrace.Span_CONSUMER:
		ocKind = OpenTracingSpanKindConsumer
	case otlptrace.Span_PRODUCER:
		ocKind = OpenTracingSpanKindProducer
	case otlptrace.Span_SPAN_KIND_UNSPECIFIED:
	case otlptrace.Span_INTERNAL:
	case otlptrace.Span_SERVER: // explicitly handled as SpanKind
	case otlptrace.Span_CLIENT: // explicitly handled as SpanKind
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
