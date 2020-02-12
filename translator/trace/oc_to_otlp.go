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

package tracetranslator

import (
	"strconv"
	"strings"

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
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

// OTLP attributes to map certain OpenCensus proto fields. These fields don't have
// corresponding fields in OTLP, nor are defined in OTLP semantic conventions.
// TODO: decide if any of these must be in OTLP semantic conventions.
const (
	ocAttributeProcessStartTime  = "opencensus.starttime"
	ocAttributeProcessID         = "opencensus.pid"
	ocAttributeExporterVersion   = "opencensus.exporterversion"
	ocAttributeResourceType      = "opencensus.resourcetype"
	ocTimeEventMessageEventType  = "opencensus.timeevent.messageevent.type"
	ocTimeEventMessageEventID    = "opencensus.timeevent.messageevent.id"
	ocTimeEventMessageEventUSize = "opencensus.timeevent.messageevent.usize"
	ocTimeEventMessageEventCSize = "opencensus.timeevent.messageevent.csize"
)

func ocToOtlp(td consumerdata.TraceData) consumerdata.OTLPTraceData {

	if td.Node == nil && td.Resource == nil && len(td.Spans) == 0 {
		return consumerdata.OTLPTraceData{}
	}

	resource := ocNodeResourceToOtlp(td.Node, td.Resource)

	resourceSpans := &otlptrace.ResourceSpans{
		Resource: resource,
	}
	resourceSpanList := []*otlptrace.ResourceSpans{resourceSpans}

	if len(td.Spans) != 0 {
		resourceSpans.Spans = make([]*otlptrace.Span, 0, len(td.Spans))

		for _, ocSpan := range td.Spans {
			if ocSpan == nil {
				// Skip nil spans.
				continue
			}

			otlpSpan := ocSpanToOtlp(ocSpan)

			if ocSpan.Resource != nil {
				// Add a separate ResourceSpans item just for this span since it
				// has a different Resource.
				separateRS := &otlptrace.ResourceSpans{
					Resource: ocNodeResourceToOtlp(nil, ocSpan.Resource),
					Spans:    []*otlptrace.Span{otlpSpan},
				}
				resourceSpanList = append(resourceSpanList, separateRS)
			} else {
				// Otherwise add the span to the first ResourceSpans item.
				resourceSpans.Spans = append(resourceSpans.Spans, otlpSpan)
			}
		}
	}

	return consumerdata.NewOTLPTraceData(resourceSpanList)
}

func timestampToUnixnano(ts *timestamp.Timestamp) uint64 {
	return uint64(internal.TimestampToTime(ts).UnixNano())
}

func ocSpanToOtlp(ocSpan *octrace.Span) *otlptrace.Span {
	attrs, droppedAttrCount := ocAttrsToOtlp(ocSpan.Attributes)
	events, droppedEventCount := ocEventsToOtlp(ocSpan.TimeEvents)
	links, droppedLinkCount := ocLinksToOtlp(ocSpan.Links)

	childSpanCount := int32(0)
	if ocSpan.ChildSpanCount != nil {
		childSpanCount = int32(ocSpan.ChildSpanCount.Value)
	}

	otlpSpan := &otlptrace.Span{
		TraceId:                ocSpan.TraceId,
		SpanId:                 ocSpan.SpanId,
		Tracestate:             ocTraceStateToOtlp(ocSpan.Tracestate),
		ParentSpanId:           ocSpan.ParentSpanId,
		Name:                   truncableStringToStr(ocSpan.Name),
		Kind:                   ocSpanKindToOtlp(ocSpan.Kind, ocSpan.Attributes),
		StartTimeUnixnano:      timestampToUnixnano(ocSpan.StartTime),
		EndTimeUnixnano:        timestampToUnixnano(ocSpan.EndTime),
		Attributes:             attrs,
		DroppedAttributesCount: droppedAttrCount,
		Events:                 events,
		DroppedEventsCount:     droppedEventCount,
		Links:                  links,
		DroppedLinksCount:      droppedLinkCount,
		Status:                 ocStatusToOtlp(ocSpan.Status),
		LocalChildSpanCount:    childSpanCount,
	}

	return otlpSpan
}

func ocStatusToOtlp(ocStatus *octrace.Status) *otlptrace.Status {
	if ocStatus == nil {
		return nil
	}
	return &otlptrace.Status{
		Code:    otlptrace.Status_StatusCode(ocStatus.Code),
		Message: ocStatus.Message,
	}
}

// Convert tracestate to W3C format. See the https://w3c.github.io/trace-context/
func ocTraceStateToOtlp(ocTracestate *octrace.Span_Tracestate) string {
	if ocTracestate == nil {
		return ""
	}
	var sb strings.Builder
	for i, entry := range ocTracestate.Entries {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(strings.Join([]string{entry.Key, entry.Value}, "="))
	}
	return sb.String()
}

func ocAttrsToOtlp(ocAttrs *octrace.Span_Attributes) (otlpAttrs []*otlpcommon.AttributeKeyValue, droppedCount uint32) {
	if ocAttrs == nil {
		return
	}

	otlpAttrs = make([]*otlpcommon.AttributeKeyValue, 0, len(ocAttrs.AttributeMap))
	for key, ocAttr := range ocAttrs.AttributeMap {

		otlpAttr := &otlpcommon.AttributeKeyValue{Key: key}

		switch attribValue := ocAttr.Value.(type) {
		case *octrace.AttributeValue_StringValue:
			otlpAttr.StringValue = truncableStringToStr(attribValue.StringValue)
			otlpAttr.Type = otlpcommon.AttributeKeyValue_STRING

		case *octrace.AttributeValue_IntValue:
			otlpAttr.IntValue = attribValue.IntValue
			otlpAttr.Type = otlpcommon.AttributeKeyValue_INT

		case *octrace.AttributeValue_BoolValue:
			otlpAttr.BoolValue = attribValue.BoolValue
			otlpAttr.Type = otlpcommon.AttributeKeyValue_BOOL

		case *octrace.AttributeValue_DoubleValue:
			otlpAttr.DoubleValue = attribValue.DoubleValue
			otlpAttr.Type = otlpcommon.AttributeKeyValue_DOUBLE

		default:
			str := "<Unknown OpenCensus Attribute>"
			otlpAttr.StringValue = str
			otlpAttr.Type = otlpcommon.AttributeKeyValue_STRING
		}
		otlpAttrs = append(otlpAttrs, otlpAttr)
	}
	droppedCount = uint32(ocAttrs.DroppedAttributesCount)
	return
}

func ocSpanKindToOtlp(ocKind octrace.Span_SpanKind, ocAttrs *octrace.Span_Attributes) otlptrace.Span_SpanKind {
	switch ocKind {
	case octrace.Span_SERVER:
		return otlptrace.Span_SERVER

	case octrace.Span_CLIENT:
		return otlptrace.Span_CLIENT

	case octrace.Span_SPAN_KIND_UNSPECIFIED:
		// Span kind field is unspecified, check if TagSpanKind attribute is set.
		// This can happen if span kind had no equivalent in OC, so we could represent it in
		// the SpanKind. In that case the span kind may be a special attribute TagSpanKind.
		if ocAttrs != nil {
			kindAttr := ocAttrs.AttributeMap[TagSpanKind]
			if kindAttr != nil {
				strVal, ok := kindAttr.Value.(*octrace.AttributeValue_StringValue)
				if ok && strVal != nil {
					var otlpKind otlptrace.Span_SpanKind
					switch OpenTracingSpanKind(truncableStringToStr(strVal.StringValue)) {
					case OpenTracingSpanKindConsumer:
						otlpKind = otlptrace.Span_CONSUMER
					case OpenTracingSpanKindProducer:
						otlpKind = otlptrace.Span_PRODUCER
					default:
						return otlptrace.Span_SPAN_KIND_UNSPECIFIED
					}
					delete(ocAttrs.AttributeMap, TagSpanKind)
					return otlpKind
				}
			}
		}
		return otlptrace.Span_SPAN_KIND_UNSPECIFIED

	default:
		return otlptrace.Span_SPAN_KIND_UNSPECIFIED
	}
}

func ocEventsToOtlp(ocEvents *octrace.Span_TimeEvents) (otlpEvents []*otlptrace.Span_Event, droppedCount uint32) {
	if ocEvents == nil {
		return
	}

	droppedCount = uint32(ocEvents.DroppedMessageEventsCount + ocEvents.DroppedAnnotationsCount)
	if len(ocEvents.TimeEvent) == 0 {
		return
	}

	otlpEvents = make([]*otlptrace.Span_Event, 0, len(ocEvents.TimeEvent))

	for _, ocEvent := range ocEvents.TimeEvent {
		if ocEvent == nil {
			continue
		}

		otlpEvent := &otlptrace.Span_Event{
			TimeUnixnano: timestampToUnixnano(ocEvent.Time),
		}

		switch teValue := ocEvent.Value.(type) {
		case *octrace.Span_TimeEvent_Annotation_:
			if teValue.Annotation != nil {
				otlpEvent.Name = truncableStringToStr(teValue.Annotation.Description)
				attrs, droppedCount := ocAttrsToOtlp(teValue.Annotation.Attributes)
				otlpEvent.Attributes = attrs
				otlpEvent.DroppedAttributesCount = droppedCount
			}

		case *octrace.Span_TimeEvent_MessageEvent_:
			otlpEvent.Attributes = ocMessageEventToOtlpAttrs(teValue.MessageEvent)

		default:
			otlpEvent.Name = "An unknown OpenCensus TimeEvent type was detected when translating to OTLP"
		}

		otlpEvents = append(otlpEvents, otlpEvent)
	}
	return
}

func ocLinksToOtlp(ocLinks *octrace.Span_Links) (otlpLinks []*otlptrace.Span_Link, droppedCount uint32) {
	if ocLinks == nil {
		return nil, 0
	}

	droppedCount = uint32(ocLinks.DroppedLinksCount)
	if len(ocLinks.Link) == 0 {
		return nil, droppedCount
	}

	otlpLinks = make([]*otlptrace.Span_Link, 0, len(ocLinks.Link))

	for _, ocLink := range ocLinks.Link {
		if ocLink == nil {
			continue
		}

		attrs, droppedCount := ocAttrsToOtlp(ocLink.Attributes)
		otlpLink := &otlptrace.Span_Link{
			TraceId:                ocLink.TraceId,
			SpanId:                 ocLink.SpanId,
			Tracestate:             ocTraceStateToOtlp(ocLink.Tracestate),
			Attributes:             attrs,
			DroppedAttributesCount: droppedCount,
		}

		otlpLinks = append(otlpLinks, otlpLink)
	}
	return otlpLinks, droppedCount
}

func ocMessageEventToOtlpAttrs(msgEvent *octrace.Span_TimeEvent_MessageEvent) []*otlpcommon.AttributeKeyValue {
	if msgEvent == nil {
		return nil
	}

	return []*otlpcommon.AttributeKeyValue{
		{
			Key:         ocTimeEventMessageEventType,
			StringValue: msgEvent.Type.String(),
			Type:        otlpcommon.AttributeKeyValue_STRING,
		},
		{
			Key:      ocTimeEventMessageEventID,
			IntValue: int64(msgEvent.Id),
			Type:     otlpcommon.AttributeKeyValue_INT,
		},
		{
			Key:      ocTimeEventMessageEventUSize,
			IntValue: int64(msgEvent.UncompressedSize),
			Type:     otlpcommon.AttributeKeyValue_INT,
		},
		{
			Key:      ocTimeEventMessageEventCSize,
			IntValue: int64(msgEvent.CompressedSize),
			Type:     otlpcommon.AttributeKeyValue_INT,
		},
	}
}

func truncableStringToStr(ts *octrace.TruncatableString) string {
	if ts == nil {
		return ""
	}
	return ts.Value
}

func ocNodeResourceToOtlp(node *occommon.Node, resource *ocresource.Resource) *otlpresource.Resource {
	otlpResource := &otlpresource.Resource{}

	// Move all special fields in Node and in Resource to a temporary attributes map.
	// After that we will build the attributes slice in OTLP format from this map.
	// This ensures there are no attributes with duplicate keys.

	// Number of special fields in the Node. See the code below that deals with special fields.
	const specialNodeAttrCount = 7

	// Number of special fields in the Resource.
	const specialResourceAttrCount = 1

	// Calculate maximum total number of attributes. It is OK if we are a bit higher than
	// the exact number since this is only needed for capacity reservation.
	totalAttrCount := 0
	if node != nil {
		totalAttrCount += len(node.Attributes) + specialNodeAttrCount
	}
	if resource != nil {
		totalAttrCount += len(resource.Labels) + specialResourceAttrCount
	}

	// Create a temporary map where we will place all attributes from the Node and Resource.
	attrs := make(map[string]string, totalAttrCount)

	if node != nil {
		// Copy all Attributes.
		for k, v := range node.Attributes {
			attrs[k] = v
		}

		// Add all special fields.
		if node.ServiceInfo != nil {
			if node.ServiceInfo.Name != "" {
				attrs[conventions.AttributeServiceName] = node.ServiceInfo.Name
			}
		}
		if node.Identifier != nil {
			if node.Identifier.StartTimestamp != nil {
				attrs[ocAttributeProcessStartTime] = ptypes.TimestampString(node.Identifier.StartTimestamp)
			}
			if node.Identifier.HostName != "" {
				attrs[conventions.AttributeHostHostname] = node.Identifier.HostName
			}
			if node.Identifier.Pid != 0 {
				attrs[ocAttributeProcessID] = strconv.Itoa(int(node.Identifier.Pid))
			}
		}
		if node.LibraryInfo != nil {
			if node.LibraryInfo.CoreLibraryVersion != "" {
				attrs[conventions.AttributeLibraryVersion] = node.LibraryInfo.CoreLibraryVersion
			}
			if node.LibraryInfo.ExporterVersion != "" {
				attrs[ocAttributeExporterVersion] = node.LibraryInfo.ExporterVersion
			}
			if node.LibraryInfo.Language != occommon.LibraryInfo_LANGUAGE_UNSPECIFIED {
				attrs[conventions.AttributeLibraryLanguage] = node.LibraryInfo.Language.String()
			}
		}
	}

	if resource != nil {
		// Copy resource Labels.
		for k, v := range resource.Labels {
			attrs[k] = v
		}
		// Add special fields.
		if resource.Type != "" {
			attrs[ocAttributeResourceType] = resource.Type
		}
	}

	// Convert everything from the temporary matp to OTLP format.
	otlpResource.Attributes = attrMapToOtlp(attrs)

	return otlpResource
}

func attrMapToOtlp(ocAttrs map[string]string) []*otlpcommon.AttributeKeyValue {
	if len(ocAttrs) == 0 {
		return nil
	}

	otlpAttrs := make([]*otlpcommon.AttributeKeyValue, len(ocAttrs))
	i := 0
	for k, v := range ocAttrs {
		otlpAttrs[i] = otlpStringAttr(k, v)
		i++
	}
	return otlpAttrs
}

func otlpStringAttr(key string, val string) *otlpcommon.AttributeKeyValue {
	return &otlpcommon.AttributeKeyValue{
		Key:         key,
		Type:        otlpcommon.AttributeKeyValue_STRING,
		StringValue: val,
	}
}
