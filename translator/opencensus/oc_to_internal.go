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

package opencensus

import (
	"strings"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	octrace "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/ptypes"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
	tracetranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
)

func ocToInternal(td consumerdata.TraceData) data.TraceData {

	if td.Node == nil && td.Resource == nil && len(td.Spans) == 0 {
		return data.TraceData{}
	}

	resource := ocNodeResourceToInternal(td.Node, td.Resource)

	resourceSpans := data.NewResourceSpans(resource, nil)
	resourceSpanList := []*data.ResourceSpans{resourceSpans}

	if len(td.Spans) == 0 {
		return data.NewTraceData(resourceSpanList)
	}

	// We may need to split OC spans into several ResourceSpans. OC Spans can have a
	// Resource field inside them set to nil which indicates they use the Resource
	// specified in "td.Resource", or they can have the Resource field inside them set
	// to non-nil which indicates they have overridden Resource field and "td.Resource"
	// does not apply to those spans.
	//
	// Each OC Span that has its own Resource field set to non-nil must be placed in a
	// separate ResourceSpans instance, containing only that span. All other OC Spans
	// that have nil Resource field must be placed in one other ResourceSpans instance,
	// which will gets its Resource field from "td.Resource".
	//
	// We will end up with with one or more ResourceSpans like this:
	//
	// ResourceSpans           ResourceSpans  ResourceSpans
	// +-----+-----+---+-----+ +------------+ +------------+
	// |Span1|Span2|...|SpanM| |Span        | |Span        | ...
	// +-----+-----+---+-----+ +------------+ +------------+

	// Count the number of spans that have nil Resource and need to be combined
	// in one slice.
	combinedSpanCount := 0
	for _, ocSpan := range td.Spans {
		if ocSpan == nil {
			// Skip nil spans.
			continue
		}
		if ocSpan.Resource == nil {
			combinedSpanCount++
		}
	}

	// Allocate a slice for spans that need to be combined into one ResourceSpans.
	combinedSpans := data.NewSpanSlice(combinedSpanCount)
	resourceSpans.SetInstrumentationLibrarySpans([]*data.InstrumentationLibrarySpans{
		data.NewInstrumentationLibrarySpans(data.NewInstrumentationLibrary(), combinedSpans)})

	// Now do the span translation and place them in appropriate ResourceSpans
	// instances.

	// Index to next available slot in "combinedSpans" slice.
	i := 0
	for _, ocSpan := range td.Spans {
		if ocSpan == nil {
			// Skip nil spans.
			continue
		}

		if ocSpan.Resource == nil {
			// Add the span to the "combinedSpans". combinedSpans length is equal
			// to combinedSpanCount. The loop above that calculates combinedSpanCount
			// has exact same conditions as we have here in this loop.
			destSpan := combinedSpans[i]
			i++

			// Convert the span.
			ocSpanToInternal(destSpan, ocSpan)
		} else {
			// This span has a different Resource and must be placed in a different
			// ResourceSpans instance. Create a separate ResourceSpans item just for this span.
			separateRS := ocSpanToResourceSpans(ocSpan, td.Node, ocSpan.Resource)

			// Add the newly created ResourceSpans to the final list that we will return.
			resourceSpanList = append(resourceSpanList, separateRS)
		}
	}

	return data.NewTraceData(resourceSpanList)
}

func ocSpanToResourceSpans(ocSpan *octrace.Span, node *occommon.Node, resource *ocresource.Resource) *data.ResourceSpans {
	destSpan := data.NewSpan()
	ocSpanToInternal(destSpan, ocSpan)

	// Create a separate ResourceSpans item just for this span.
	return data.NewResourceSpans(
		ocNodeResourceToInternal(node, resource),
		[]*data.InstrumentationLibrarySpans{
			data.NewInstrumentationLibrarySpans(data.NewInstrumentationLibrary(), []*data.Span{destSpan})},
	)
}

func ocSpanToInternal(dest *data.Span, src *octrace.Span) {
	events, droppedEventCount := ocEventsToInternal(src.TimeEvents)
	links, droppedLinkCount := ocLinksToInternal(src.Links)

	// Note that ocSpanKindToInternal must be called before ocAttrsToInternal
	// since it may modify src.Attributes (remove the attribute which represents the
	// span kind).
	kind := ocSpanKindToInternal(src.Kind, src.Attributes)
	attrs := ocAttrsToInternal(src.Attributes)

	dest.SetTraceID(data.NewTraceID(src.TraceId))
	dest.SetSpanID(data.NewSpanID(src.SpanId))
	dest.SetTraceState(ocTraceStateToInternal(src.Tracestate))
	dest.SetParentSpanID(data.NewSpanID(src.ParentSpanId))
	dest.SetName(src.Name.GetValue())
	dest.SetKind(kind)
	dest.SetStartTime(internal.TimestampToUnixnano(src.StartTime))
	dest.SetEndTime(internal.TimestampToUnixnano(src.EndTime))
	dest.SetAttributes(attrs)
	dest.SetEvents(events)
	dest.SetDroppedEventsCount(droppedEventCount)
	dest.SetLinks(links)
	dest.SetDroppedLinksCount(droppedLinkCount)
	dest.SetStatus(ocStatusToInternal(src.Status))
}

func ocStatusToInternal(ocStatus *octrace.Status) data.SpanStatus {
	if ocStatus == nil {
		return data.SpanStatus{}
	}
	return data.NewSpanStatus(data.StatusCode(ocStatus.Code), ocStatus.Message)
}

// Convert tracestate to W3C format. See the https://w3c.github.io/trace-context/
func ocTraceStateToInternal(ocTracestate *octrace.Span_Tracestate) data.TraceState {
	if ocTracestate == nil {
		return ""
	}
	var sb strings.Builder
	for i, entry := range ocTracestate.Entries {
		sb.Grow(1 + len(entry.Key) + 1 + len(entry.Value))
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(entry.Key)
		sb.WriteString("=")
		sb.WriteString(entry.Value)
	}
	return data.TraceState(sb.String())
}

func ocAttrsToInternal(ocAttrs *octrace.Span_Attributes) data.Attributes {
	if ocAttrs == nil {
		return data.Attributes{}
	}

	attrCount := len(ocAttrs.AttributeMap)
	attrMap := make(data.AttributesMap, attrCount)
	if attrCount > 0 {
		values := data.NewAttributeValueSlice(attrCount)
		i := 0
		for key, ocAttr := range ocAttrs.AttributeMap {
			switch attribValue := ocAttr.Value.(type) {
			case *octrace.AttributeValue_StringValue:
				values[i].SetString(attribValue.StringValue.GetValue())

			case *octrace.AttributeValue_IntValue:
				values[i].SetInt(attribValue.IntValue)

			case *octrace.AttributeValue_BoolValue:
				values[i].SetBool(attribValue.BoolValue)

			case *octrace.AttributeValue_DoubleValue:
				values[i].SetDouble(attribValue.DoubleValue)

			default:
				str := "<Unknown OpenCensus attribute value type>"
				values[i].SetString(str)
			}
			attrMap[key] = values[i]
			i++
		}
	}
	droppedCount := uint32(ocAttrs.DroppedAttributesCount)
	return data.NewAttributes(attrMap, droppedCount)
}

func ocSpanKindToInternal(ocKind octrace.Span_SpanKind, ocAttrs *octrace.Span_Attributes) data.SpanKind {
	switch ocKind {
	case octrace.Span_SERVER:
		return data.SpanKindSERVER

	case octrace.Span_CLIENT:
		return data.SpanKindCLIENT

	case octrace.Span_SPAN_KIND_UNSPECIFIED:
		// Span kind field is unspecified, check if TagSpanKind attribute is set.
		// This can happen if span kind had no equivalent in OC, so we could represent it in
		// the SpanKind. In that case the span kind may be a special attribute TagSpanKind.
		if ocAttrs != nil {
			kindAttr := ocAttrs.AttributeMap[tracetranslator.TagSpanKind]
			if kindAttr != nil {
				strVal, ok := kindAttr.Value.(*octrace.AttributeValue_StringValue)
				if ok && strVal != nil {
					var otlpKind data.SpanKind
					switch tracetranslator.OpenTracingSpanKind(strVal.StringValue.GetValue()) {
					case tracetranslator.OpenTracingSpanKindConsumer:
						otlpKind = data.SpanKindCONSUMER
					case tracetranslator.OpenTracingSpanKindProducer:
						otlpKind = data.SpanKindPRODUCER
					default:
						return data.SpanKindUNSPECIFIED
					}
					delete(ocAttrs.AttributeMap, tracetranslator.TagSpanKind)
					return otlpKind
				}
			}
		}
		return data.SpanKindUNSPECIFIED

	default:
		return data.SpanKindUNSPECIFIED
	}
}

func ocEventsToInternal(ocEvents *octrace.Span_TimeEvents) (events []*data.SpanEvent, droppedCount uint32) {
	if ocEvents == nil {
		return
	}

	droppedCount = uint32(ocEvents.DroppedMessageEventsCount + ocEvents.DroppedAnnotationsCount)

	eventCount := len(ocEvents.TimeEvent)
	if eventCount == 0 {
		return
	}

	events = data.NewSpanEventSlice(eventCount)

	i := 0
	for _, ocEvent := range ocEvents.TimeEvent {
		if ocEvent == nil {
			// Skip nil source events.
			continue
		}

		event := events[i]
		i++

		event.SetTimestamp(internal.TimestampToUnixnano(ocEvent.Time))

		switch teValue := ocEvent.Value.(type) {
		case *octrace.Span_TimeEvent_Annotation_:
			if teValue.Annotation != nil {
				event.SetName(teValue.Annotation.Description.GetValue())
				attrs := ocAttrsToInternal(teValue.Annotation.Attributes)
				event.SetAttributes(attrs)
			}

		case *octrace.Span_TimeEvent_MessageEvent_:
			event.SetAttributes(ocMessageEventToInternalAttrs(teValue.MessageEvent))

		default:
			event.SetName("An unknown OpenCensus TimeEvent type was detected when translating")
		}
	}

	// Truncate the slice to only include populated items.
	events = events[0:i]

	return
}

func ocLinksToInternal(ocLinks *octrace.Span_Links) (links []*data.SpanLink, droppedCount uint32) {
	if ocLinks == nil {
		return
	}

	droppedCount = uint32(ocLinks.DroppedLinksCount)

	linkCount := len(ocLinks.Link)
	if linkCount == 0 {
		return
	}

	links = data.NewSpanLinkSlice(linkCount)

	i := 0
	for _, ocLink := range ocLinks.Link {
		if ocLink == nil {
			continue
		}

		link := links[i]
		i++

		link.SetTraceID(data.NewTraceID(ocLink.TraceId))
		link.SetSpanID(data.NewSpanID(ocLink.SpanId))
		link.SetTraceState(ocTraceStateToInternal(ocLink.Tracestate))
		link.SetAttributes(ocAttrsToInternal(ocLink.Attributes))
	}

	// Truncate the slice to only include populated items.
	links = links[0:i]

	return
}

func ocMessageEventToInternalAttrs(msgEvent *octrace.Span_TimeEvent_MessageEvent) data.Attributes {
	if msgEvent == nil {
		return data.Attributes{}
	}

	attrs := data.NewAttributeValueSlice(4)
	attrs[0].SetString(msgEvent.Type.String())
	attrs[1].SetInt(int64(msgEvent.Id))
	attrs[2].SetInt(int64(msgEvent.UncompressedSize))
	attrs[3].SetInt(int64(msgEvent.CompressedSize))

	return data.NewAttributes(
		map[string]data.AttributeValue{
			conventions.OCTimeEventMessageEventType:  attrs[0],
			conventions.OCTimeEventMessageEventID:    attrs[1],
			conventions.OCTimeEventMessageEventUSize: attrs[2],
			conventions.OCTimeEventMessageEventCSize: attrs[3],
		},
		0)
}

func ocNodeResourceToInternal(ocNode *occommon.Node, ocResource *ocresource.Resource) data.Resource {
	resource := data.NewResource()

	// Number of special fields in the Node. See the code below that deals with special fields.
	const specialNodeAttrCount = 7

	// Number of special fields in the Resource.
	const specialResourceAttrCount = 1

	// Calculate maximum total number of attributes. It is OK if we are a bit higher than
	// the exact number since this is only needed for capacity reservation.
	maxTotalAttrCount := 0
	if ocNode != nil {
		maxTotalAttrCount += len(ocNode.Attributes) + specialNodeAttrCount
	}
	if ocResource != nil {
		maxTotalAttrCount += len(ocResource.Labels) + specialResourceAttrCount
	}

	// Create a map where we will place all attributes from the Node and Resource.
	attrs := make(data.AttributesMap, maxTotalAttrCount)

	if ocNode != nil {
		// Copy all Attributes.
		for k, v := range ocNode.Attributes {
			attrs[k] = data.NewAttributeValueString(v)
		}

		// Add all special fields.
		if ocNode.ServiceInfo != nil {
			if ocNode.ServiceInfo.Name != "" {
				attrs[conventions.AttributeServiceName] = data.NewAttributeValueString(
					ocNode.ServiceInfo.Name)
			}
		}
		if ocNode.Identifier != nil {
			if ocNode.Identifier.StartTimestamp != nil {
				attrs[conventions.OCAttributeProcessStartTime] = data.NewAttributeValueString(
					ptypes.TimestampString(ocNode.Identifier.StartTimestamp))
			}
			if ocNode.Identifier.HostName != "" {
				attrs[conventions.AttributeHostHostname] = data.NewAttributeValueString(
					ocNode.Identifier.HostName)
			}
			if ocNode.Identifier.Pid != 0 {
				attrs[conventions.OCAttributeProcessID] = data.NewAttributeValueInt(int64(ocNode.Identifier.Pid))
			}
		}
		if ocNode.LibraryInfo != nil {
			if ocNode.LibraryInfo.CoreLibraryVersion != "" {
				attrs[conventions.AttributeLibraryVersion] = data.NewAttributeValueString(
					ocNode.LibraryInfo.CoreLibraryVersion)
			}
			if ocNode.LibraryInfo.ExporterVersion != "" {
				attrs[conventions.OCAttributeExporterVersion] = data.NewAttributeValueString(
					ocNode.LibraryInfo.ExporterVersion)
			}
			if ocNode.LibraryInfo.Language != occommon.LibraryInfo_LANGUAGE_UNSPECIFIED {
				attrs[conventions.AttributeLibraryLanguage] = data.NewAttributeValueString(
					ocNode.LibraryInfo.Language.String())
			}
		}
	}

	if ocResource != nil {
		// Copy resource Labels.
		for k, v := range ocResource.Labels {
			attrs[k] = data.NewAttributeValueString(v)
		}
		// Add special fields.
		if ocResource.Type != "" {
			attrs[conventions.OCAttributeResourceType] = data.NewAttributeValueString(ocResource.Type)
		}
	}

	if len(attrs) != 0 {
		// TODO: Re-evaluate if we want to construct a map first, or we can construct directly
		// a slice of AttributeKeyValue.
		resource.SetAttributes(data.NewAttributeMap(attrs))
	}

	return resource
}
